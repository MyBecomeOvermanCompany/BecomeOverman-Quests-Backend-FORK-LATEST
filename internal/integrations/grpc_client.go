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

func (c *RecommendationGRPCClient) AddUsers(ctx context.Context, users []*pb.UserWithQuestIDs) (*pb.AddUsersResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	resp, err := c.client.AddUsers(ctx, &pb.AddUsersRequest{
		Users: users,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC AddUsers failed: %w", err)
	}
	return resp, nil
}

// RecommendUsers — рекомендация похожих пользователей
func (c *RecommendationGRPCClient) RecommendUsers(ctx context.Context, userID int32, topK int32) (*pb.RecommendUsersResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := c.client.RecommendUsers(ctx, &pb.RecommendUsersRequest{
		UserId: userID,
		TopK:   topK,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC RecommendUsers failed: %w", err)
	}

	return resp, nil
}

// RecommendQuests — рекомендация квестов для пользователя
func (c *RecommendationGRPCClient) RecommendQuests(ctx context.Context, userQuestIDs []int32, topK int32, category string) (*pb.RecommendQuestsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := c.client.RecommendQuests(ctx, &pb.RecommendQuestsRequest{
		UserQuestIds: userQuestIDs,
		TopK:         topK,
		Category:     category,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC RecommendQuests failed: %w", err)
	}

	return resp, nil
}
