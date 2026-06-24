package client

import (
	"context"
	"log"
	"time"

	pb "go-deployment1/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCClient struct {
	addr string
}

func NewGRPCClient(addr string) *GRPCClient {
	return &GRPCClient{addr: addr}
}

func (c *GRPCClient) SendPrediction(homeTeam, awayTeam string, homeGoals, awayGoals int32, username, timestamp string) error {
	conn, err := grpc.NewClient(
		c.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pb.NewPredictionServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.PredictionRequest{
		HomeTeam:  homeTeam,
		AwayTeam:  awayTeam,
		HomeGoals: homeGoals,
		AwayGoals: awayGoals,
		Username:  username,
		Timestamp: timestamp,
	}

	resp, err := client.SendPrediction(ctx, req)
	if err != nil {
		return err
	}

	log.Printf("Respuesta del gRPC Server: %s", resp.Message)
	return nil
}
