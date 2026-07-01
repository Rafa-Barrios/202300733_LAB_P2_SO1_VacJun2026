package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/valkey-io/valkey-go"
)

type Prediction struct {
	HomeTeam  string `json:"home_team"`
	AwayTeam  string `json:"away_team"`
	HomeGoals int32  `json:"home_goals"`
	AwayGoals int32  `json:"away_goals"`
	Username  string `json:"username"`
	Timestamp string `json:"timestamp"`
}

func connectRabbitMQ(url string) (*amqp.Connection, *amqp.Channel, error) {
	var conn *amqp.Connection
	var err error
	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Printf("Intento %d: esperando RabbitMQ... %v", i+1, err)
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		return nil, nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, nil, err
	}
	_, err = ch.QueueDeclare("predictions", true, false, false, false, nil)
	if err != nil {
		return nil, nil, err
	}
	return conn, ch, nil
}

func connectValkey(addr string) (valkey.Client, error) {
	var client valkey.Client
	var err error
	for i := 0; i < 10; i++ {
		client, err = valkey.NewClient(valkey.ClientOption{
			InitAddress: []string{addr},
		})
		if err == nil {
			break
		}
		log.Printf("Intento %d: esperando Valkey... %v", i+1, err)
		time.Sleep(5 * time.Second)
	}
	return client, err
}

func storePrediction(ctx context.Context, client valkey.Client, pred Prediction) error {
	// Total predicciones general
	client.Do(ctx, client.B().Incr().Key("total:predictions").Build())

	// Predicciones del equipo MEX
	if pred.HomeTeam == "MEX" || pred.AwayTeam == "MEX" {
		client.Do(ctx, client.B().Incr().Key("mex:total").Build())
	}

	// Goles como local de MEX
	if pred.HomeTeam == "MEX" {
		golesStr := fmt.Sprintf("%d", pred.HomeGoals)
		client.Do(ctx, client.B().Lpush().Key("mex:home_goals").Element(golesStr).Build())
		client.Do(ctx, client.B().Zadd().Key("mex:home_goals_sorted").ScoreMember().
			ScoreMember(float64(pred.HomeGoals), golesStr+":"+pred.Timestamp).Build())
		// Serie temporal local
		score := float64(time.Now().Unix())
		client.Do(ctx, client.B().Zadd().Key("mex:home_goals_ts").ScoreMember().
			ScoreMember(score, fmt.Sprintf("%d:%d", pred.HomeGoals, time.Now().UnixNano())).Build())
	}

	// Goles como visitante de MEX
	if pred.AwayTeam == "MEX" {
		golesStr := fmt.Sprintf("%d", pred.AwayGoals)
		client.Do(ctx, client.B().Lpush().Key("mex:away_goals").Element(golesStr).Build())
		client.Do(ctx, client.B().Zadd().Key("mex:away_goals_sorted").ScoreMember().
			ScoreMember(float64(pred.AwayGoals), golesStr+":"+pred.Timestamp).Build())
		// Serie temporal visitante
		score := float64(time.Now().Unix())
		client.Do(ctx, client.B().Zadd().Key("mex:away_goals_ts").ScoreMember().
			ScoreMember(score, fmt.Sprintf("%d:%d", pred.AwayGoals, time.Now().UnixNano())).Build())
	}

	// Actividad por usuario
	client.Do(ctx, client.B().Zincrby().Key("users:activity").
		Increment(1).Member(pred.Username).Build())

	// Serie temporal MEX
	if pred.HomeTeam == "MEX" || pred.AwayTeam == "MEX" {
		score := float64(time.Now().Unix())
		client.Do(ctx, client.B().Zadd().Key("mex:timeline").ScoreMember().
			ScoreMember(score, pred.Timestamp).Build())
	}

	// Victorias por equipo
	if pred.HomeGoals > pred.AwayGoals {
		client.Do(ctx, client.B().Zincrby().Key("teams:wins").
			Increment(1).Member(pred.HomeTeam).Build())
	} else if pred.AwayGoals > pred.HomeGoals {
		client.Do(ctx, client.B().Zincrby().Key("teams:wins").
			Increment(1).Member(pred.AwayTeam).Build())
	}

	// Moda de goles local MEX
	if pred.HomeTeam == "MEX" {
		client.Do(ctx, client.B().Zincrby().Key("mex:home_goals_moda").
			Increment(1).Member(fmt.Sprintf("%d", pred.HomeGoals)).Build())
	}

	// Moda de goles visitante MEX
	if pred.AwayTeam == "MEX" {
		client.Do(ctx, client.B().Zincrby().Key("mex:away_goals_moda").
			Increment(1).Member(fmt.Sprintf("%d", pred.AwayGoals)).Build())
	}

	log.Printf("Almacenado en Valkey: %s vs %s | %d-%d | user: %s",
		pred.HomeTeam, pred.AwayTeam, pred.HomeGoals, pred.AwayGoals, pred.Username)
	return nil
}

func main() {
	rmqURL := os.Getenv("RABBITMQ_URL")
	if rmqURL == "" {
		rmqURL = "amqp://admin:admin123@rabbitmq.sopes1.svc.cluster.local:5672/"
	}

	valkeyAddr := os.Getenv("VALKEY_ADDR")
	if valkeyAddr == "" {
		valkeyAddr = "valkey-service.sopes1.svc.cluster.local:6379"
	}

	log.Println("Iniciando Consumer...")

	conn, ch, err := connectRabbitMQ(rmqURL)
	if err != nil {
		log.Fatalf("Error conectando a RabbitMQ: %v", err)
	}
	defer conn.Close()
	defer ch.Close()

	valkeyClient, err := connectValkey(valkeyAddr)
	if err != nil {
		log.Fatalf("Error conectando a Valkey: %v", err)
	}
	defer valkeyClient.Close()

	msgs, err := ch.Consume("predictions", "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Error consumiendo cola: %v", err)
	}

	log.Println("Consumer listo, esperando mensajes...")

	ctx := context.Background()
	for msg := range msgs {
		var pred Prediction
		if err := json.Unmarshal(msg.Body, &pred); err != nil {
			log.Printf("Error deserializando: %v", err)
			continue
		}
		storePrediction(ctx, valkeyClient, pred)
	}
}
