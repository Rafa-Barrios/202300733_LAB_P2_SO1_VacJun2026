package rabbitmq

import (
	"context"
	"encoding/json"
	"log"
	"time"

	pb "go-deployment2/proto"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Writer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
}

type PredictionMessage struct {
	HomeTeam  string `json:"home_team"`
	AwayTeam  string `json:"away_team"`
	HomeGoals int32  `json:"home_goals"`
	AwayGoals int32  `json:"away_goals"`
	Username  string `json:"username"`
	Timestamp string `json:"timestamp"`
}

func NewWriter(url, queueName string) (*Writer, error) {
	var conn *amqp.Connection
	var err error

	// Reintentamos la conexión hasta 10 veces porque RabbitMQ puede tardar en arrancar
	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Printf("Intento %d: esperando RabbitMQ... %v", i+1, err)
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// Declaramos la cola
	_, err = ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return nil, err
	}

	log.Printf("Conectado a RabbitMQ, cola '%s' lista", queueName)

	return &Writer{
		conn:    conn,
		channel: ch,
		queue:   queueName,
	}, nil
}

func (w *Writer) Publish(req *pb.PredictionRequest) error {
	msg := PredictionMessage{
		HomeTeam:  req.HomeTeam,
		AwayTeam:  req.AwayTeam,
		HomeGoals: req.HomeGoals,
		AwayGoals: req.AwayGoals,
		Username:  req.Username,
		Timestamp: req.Timestamp,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = w.channel.PublishWithContext(ctx,
		"",      // exchange
		w.queue, // routing key
		false,   // mandatory
		false,   // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)

	if err != nil {
		return err
	}

	log.Printf("Mensaje publicado en RabbitMQ: %s vs %s", req.HomeTeam, req.AwayTeam)
	return nil
}

func (w *Writer) Close() {
	if w.channel != nil {
		w.channel.Close()
	}
	if w.conn != nil {
		w.conn.Close()
	}
}
