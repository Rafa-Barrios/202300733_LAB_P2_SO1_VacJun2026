package main

import (
	"context"
	"encoding/json"
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

	_, err = ch.QueueDeclare(
		"predictions",
		true,
		false,
		false,
		false,
		nil,
	)
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
	// Incrementamos contador total de predicciones
	err := client.Do(ctx, client.B().Incr().Key("total:predictions").Build()).Error()
	if err != nil {
		return err
	}

	// Guardamos predicciones por equipo MEX
	if pred.HomeTeam == "MEX" || pred.AwayTeam == "MEX" {
		err = client.Do(ctx, client.B().Incr().Key("mex:total").Build()).Error()
		if err != nil {
			return err
		}
	}

	// Guardamos goles como local de MEX
	if pred.HomeTeam == "MEX" {
		client.Do(ctx, client.B().Lpush().Key("mex:home_goals").Element(
			string(rune(pred.HomeGoals+'0')),
		).Build())
		client.Do(ctx, client.B().Incr().Key("mex:home:wins:"+pred.HomeTeam).Build())
	}

	// Guardamos goles como visitante de MEX
	if pred.AwayTeam == "MEX" {
		client.Do(ctx, client.B().Lpush().Key("mex:away_goals").Element(
			string(rune(pred.AwayGoals+'0')),
		).Build())
	}

	// Guardamos actividad por usuario
	client.Do(ctx, client.B().Zincrby().Key("users:activity").Increment(1).Member(pred.Username).Build())

	// Serie temporal — guardamos timestamp de cada predicción de MEX
	if pred.HomeTeam == "MEX" || pred.AwayTeam == "MEX" {
		score := float64(time.Now().Unix())
		client.Do(ctx, client.B().Zadd().Key("mex:timeline").ScoreMember().ScoreMember(score, pred.Timestamp).Build())
	}

	// Guardamos victorias por equipo (si MEX ganó como local)
	if pred.HomeTeam == "MEX" && pred.HomeGoals > pred.AwayGoals {
		client.Do(ctx, client.B().Zincrby().Key("teams:wins").Increment(1).Member("MEX").Build())
	}

	log.Printf("Predicción almacenada en Valkey: %s vs %s | %d-%d | user: %s",
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
	log.Printf("RabbitMQ: %s", rmqURL)
	log.Printf("Valkey: %s", valkeyAddr)

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

	msgs, err := ch.Consume(
		"predictions",
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Error consumiendo cola: %v", err)
	}

	log.Println("Consumer listo, esperando mensajes...")

	ctx := context.Background()
	for msg := range msgs {
		var pred Prediction
		if err := json.Unmarshal(msg.Body, &pred); err != nil {
			log.Printf("Error deserializando mensaje: %v", err)
			continue
		}

		if err := storePrediction(ctx, valkeyClient, pred); err != nil {
			log.Printf("Error almacenando en Valkey: %v", err)
		}
	}
}
