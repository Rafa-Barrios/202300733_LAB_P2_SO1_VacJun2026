package main

import (
	"context"
	"log"
	"net"
	"os"

	pb "go-deployment2/proto"
	"go-deployment2/src/rabbitmq"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedPredictionServiceServer
	rmqWriter *rabbitmq.Writer
}

func (s *server) SendPrediction(ctx context.Context, req *pb.PredictionRequest) (*pb.PredictionResponse, error) {
	log.Printf("gRPC recibido: %s vs %s | %d-%d | user: %s",
		req.HomeTeam, req.AwayTeam, req.HomeGoals, req.AwayGoals, req.Username)

	err := s.rmqWriter.Publish(req)
	if err != nil {
		log.Printf("Error publicando en RabbitMQ: %v", err)
		return &pb.PredictionResponse{
			Success: false,
			Message: "Error publicando en RabbitMQ",
		}, nil
	}

	return &pb.PredictionResponse{
		Success: true,
		Message: "Predicción publicada en RabbitMQ correctamente",
	}, nil
}

func main() {
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	rmqURL := os.Getenv("RABBITMQ_URL")
	if rmqURL == "" {
		rmqURL = "amqp://admin:admin123@rabbitmq.sopes1.svc.cluster.local:5672/"
	}

	rmqWriter, err := rabbitmq.NewWriter(rmqURL, "predictions")
	if err != nil {
		log.Fatalf("Error conectando a RabbitMQ: %v", err)
	}
	defer rmqWriter.Close()

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Error iniciando listener: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterPredictionServiceServer(grpcServer, &server{rmqWriter: rmqWriter})

	log.Printf("gRPC Server escuchando en puerto %s", grpcPort)
	log.Printf("RabbitMQ URL: %s", rmqURL)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Error sirviendo gRPC: %v", err)
	}
}
