package integrations

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pb "BecomeOverMan/internal/generated/recommendation"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// RecommendationGRPCClient — gRPC-клиент для сервиса рекомендаций
type RecommendationGRPCClient struct {
	conn   *grpc.ClientConn
	client pb.RecommendationServiceClient
}

// NewRecommendationGRPCClient создаёт новое gRPC-подключение
func NewRecommendationGRPCClient(addr string) (*RecommendationGRPCClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to recommendation gRPC service at %s: %w", addr, err)
	}

	client := pb.NewRecommendationServiceClient(conn)
	slog.Info("Connected to recommendation gRPC service", "addr", addr)

	return &RecommendationGRPCClient{
		conn:   conn,
		client: client,
	}, nil
}

// Close закрывает gRPC-соединение
func (c *RecommendationGRPCClient) Close() error {
	if c.conn != nil {
		slog.Info("Closing recommendation gRPC connection")
		return c.conn.Close()
	}
	return nil
}

// SearchQuests — семантический поиск квестов
func (c *RecommendationGRPCClient) SearchQuests(ctx context.Context, query string, topK int32, category string) (*pb.SearchQuestsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := c.client.SearchQuests(ctx, &pb.SearchQuestsRequest{
		Query:    query,
		TopK:     topK,
		Category: category,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC SearchQuests failed: %w", err)
	}

	return resp, nil
}

// HealthCheck — проверка здоровья сервиса
func (c *RecommendationGRPCClient) HealthCheck(ctx context.Context) (*pb.HealthCheckResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.HealthCheck(ctx, &pb.HealthCheckRequest{})
	if err != nil {
		return nil, fmt.Errorf("gRPC HealthCheck failed: %w", err)
	}

	return resp, nil
}
