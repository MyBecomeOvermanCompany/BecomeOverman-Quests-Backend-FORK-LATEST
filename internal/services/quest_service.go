package services

import (
	"BecomeOverMan/internal/integrations"
	"BecomeOverMan/internal/models"
	"BecomeOverMan/internal/repositories"
	"context"
	"fmt"
	"log/slog"
	"slices"

	pb "BecomeOverMan/internal/generated/recommendation"
)

type QuestService struct {
	questRepo  *repositories.QuestRepository
	userRepo   *repositories.UserRepository
	grpcClient *integrations.RecommendationGRPCClient
}

func NewQuestService(
	questRepo *repositories.QuestRepository,
	userRepo *repositories.UserRepository,
	grpcClient *integrations.RecommendationGRPCClient,
) *QuestService {
	return &QuestService{questRepo: questRepo, userRepo: userRepo, grpcClient: grpcClient}
}

func (s *QuestService) GetAvailableQuests(ctx context.Context, userID int) ([]models.Quest, error) {
	return s.questRepo.GetAvailableQuests(ctx, userID)
}

func (s *QuestService) GetQuestShop(ctx context.Context, userID int) ([]models.Quest, error) {
	return s.questRepo.GetQuestShop(ctx, userID)
}

func (s *QuestService) GetMyActiveQuests(ctx context.Context, userID int) ([]models.Quest, error) {
	return s.questRepo.GetMyActiveQuests(ctx, userID)
}

func (s *QuestService) GetMyCompletedQuests(ctx context.Context, userID int) ([]models.Quest, error) {
	return s.questRepo.GetMyCompletedQuests(ctx, userID)
}

func (s *QuestService) GetMyAllQuestsWithDetails(ctx context.Context, userID int) ([]models.Quest, error) {
	return s.questRepo.GetMyAllQuestsWithDetails(ctx, userID)
}

// PurchaseQuest handles the purchase of a quest by a user
func (s *QuestService) PurchaseQuest(ctx context.Context, userID, questID int) error {
	err := s.questRepo.PurchaseQuest(ctx, userID, questID)
	if err != nil {
		slog.Error("Failed to purchase quest", "error", err)
		return err
	}

	go func() {
		questIDS, err := s.getUserQuestIDs(userID)
		if err != nil {
			slog.Error("Failed to get user quest IDs", "error", err, "user_id", userID)
			return
		}

		if len(questIDS) == 0 {
			slog.Info("User has no quests", "user_id", userID)
		}

		// gRPC вызов вместо HTTP
		questIDsInt32 := make([]int32, len(questIDS))
		for i, id := range questIDS {
			questIDsInt32[i] = int32(id)
		}

		response, err := s.grpcClient.AddUsers(context.Background(), []*pb.UserWithQuestIDs{
			{
				UserId:   int32(userID),
				QuestIds: questIDsInt32,
			},
		})
		if err != nil {
			slog.Error("Failed to send user quest to recommendation service via gRPC", "error", err, "user_id", userID)
			return
		}

		slog.Info("User quest sent to recommendation service via gRPC", "user_id", userID, "response_status", response.Status)
	}()

	return nil
}

func (s *QuestService) getUserQuestIDs(userID int) ([]int, error) {
	return s.questRepo.GetUserQuestIDs(userID)
}

func (s *QuestService) StartQuest(ctx context.Context, userID, questID int) error {
	return s.questRepo.StartQuest(ctx, userID, questID)
}

func (s *QuestService) CompleteTask(ctx context.Context, userID, questID, taskID int) error {
	return s.questRepo.CompleteTask(ctx, userID, questID, taskID)
}

func (s *QuestService) CompleteQuest(ctx context.Context, userID, questID int) error {
	return s.questRepo.CompleteQuest(ctx, userID, questID)
}

func (s *QuestService) GetQuestDetails(ctx context.Context, questID int, userID int) (*models.Quest, error) {
	return s.questRepo.GetQuestDetails(ctx, questID, userID)
}

func (s *QuestService) CreateSharedQuest(user1ID, user2ID, questID int) error {
	return s.questRepo.CreateSharedQuest(user1ID, user2ID, questID)
}

func (s *QuestService) SaveQuestToDB(quest *models.Quest, tasks []models.Task) (int, error) {
	return s.questRepo.SaveQuestToDB(quest, tasks)
}

// SearchQuests — семантический поиск через gRPC
func (s *QuestService) SearchQuests(
	ctx context.Context,
	req models.RecommendationService_SearchQuest_Request,
	userID int,
) (models.SearchQuestsResponse, error) {
	grpcResp, err := s.grpcClient.SearchQuests(ctx, req.Query, int32(req.TopK), req.Category)
	if err != nil {
		return nil, fmt.Errorf("gRPC SearchQuests error: %w", err)
	}

	response := models.RecommendationService_SearchQuests_Response{
		Results: make([]models.RecommendationService_SearchQuest_Result, len(grpcResp.Results)),
	}
	for i, r := range grpcResp.Results {
		response.Results[i] = models.RecommendationService_SearchQuest_Result{
			ID:              int(r.Id),
			Title:           r.Title,
			Description:     r.Description,
			Category:        r.Category,
			SimilarityScore: r.SimilarityScore,
		}
	}

	questsIDS := make([]int, len(response.Results))
	for i, result := range response.Results {
		questsIDS[i] = result.ID
	}

	questsWithDetails, err := s.questRepo.SearchQuestsWithDetailsByIDs(ctx, questsIDS)
	if err != nil {
		slog.ErrorContext(ctx, "ошибка получения квестов из БД", "error", err, "ids", questsIDS)
		return nil, fmt.Errorf("внутренняя ошибка поиска: %w", err)
	}

	return models.NewSearchQuestsResponse(questsWithDetails, response), nil
}

func (s *QuestService) RecommendFriends(
	ctx context.Context,
	req models.RecommendationService_RecommendUsers_Request,
) ([]models.UserProfileWithSimilarityScore, error) {
	// gRPC вызов вместо HTTP
	grpcResp, err := s.grpcClient.RecommendUsers(ctx, int32(req.UserID), int32(req.TopK))
	if err != nil {
		return nil, fmt.Errorf("gRPC RecommendUsers error: %w", err)
	}

	userIDs := make([]int, len(grpcResp.Results))
	userIDsWithSimilarityScore := make(map[int]float64, len(grpcResp.Results))
	explanations := make(map[int]map[string]any, len(grpcResp.Results))
	for i, result := range grpcResp.Results {
		uid := int(result.UserId)
		userIDs[i] = uid
		userIDsWithSimilarityScore[uid] = result.SimilarityScore

		explanation := make(map[string]any)
		if result.Explanation != nil {
			explanation["summary"] = result.Explanation.Summary
			if result.Explanation.Details != nil {
				details := make(map[string]any)
				details["common_quests_count"] = result.Explanation.Details.CommonQuestsCount
				details["common_quests_ids"] = result.Explanation.Details.CommonQuestsIds
				details["common_categories"] = result.Explanation.Details.CommonCategories
				details["user_categories_top"] = result.Explanation.Details.UserCategoriesTop
				details["other_user_categories_top"] = result.Explanation.Details.OtherUserCategoriesTop
				details["user_quests_count"] = result.Explanation.Details.UserQuestsCount
				details["other_user_quests_count"] = result.Explanation.Details.OtherUserQuestsCount
				details["similarity_level"] = result.Explanation.Details.SimilarityLevel
				explanation["details"] = details
			}
		}
		explanations[uid] = explanation
	}

	recommendedProfiles, err := s.userRepo.GetProfiles(userIDs)
	if err != nil {
		slog.ErrorContext(ctx, "ошибка получения профилей из БД", "error", err, "ids", userIDs)
		return nil, fmt.Errorf("внутренняя ошибка рекомендации друзей: %w", err)
	}

	friendsIDS, err := s.userRepo.GetAllAcceptedFriends(req.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "ошибка получения друзей из БД", "error", err, "user_id", req.UserID)
		return nil, fmt.Errorf("внутренняя ошибка рекомендации друзей: %w", err)
	}

	recommendedNotFriendsProfiles := make([]models.UserProfile, 0, len(recommendedProfiles))
	for _, profile := range recommendedProfiles {
		if !slices.Contains(friendsIDS, profile.ID) {
			recommendedNotFriendsProfiles = append(recommendedNotFriendsProfiles, profile)
		}
	}

	result := make([]models.UserProfileWithSimilarityScore, 0, len(recommendedNotFriendsProfiles))
	for _, profile := range recommendedNotFriendsProfiles {
		result = append(result, models.UserProfileWithSimilarityScore{
			UserProfile:     profile,
			SimilarityScore: userIDsWithSimilarityScore[profile.ID],
			Explanation:     explanations[profile.ID],
		})
	}

	return result, nil
}

func (s *QuestService) RecommendQuests(ctx context.Context, userID int) (*models.RecommendationService_RecommendQuests_Resp, error) {
	questIDS, err := s.questRepo.GetUserQuestIDs(userID)
	if err != nil {
		return nil, fmt.Errorf("error getting user quest ids (while recommend quests): %w", err)
	}

	if len(questIDS) == 0 {
		return &models.RecommendationService_RecommendQuests_Resp{}, nil
	}

	questIDsInt32 := make([]int32, len(questIDS))
	for i, id := range questIDS {
		questIDsInt32[i] = int32(id)
	}

	// gRPC вызов
	grpcResp, err := s.grpcClient.RecommendQuests(ctx, questIDsInt32, 0, "")
	if err != nil {
		return nil, fmt.Errorf("gRPC RecommendQuests error: %w", err)
	}

	recommendations := make([]models.RecommendQuests_Result, len(grpcResp.Recommendations))
	for i, r := range grpcResp.Recommendations {
		recommendations[i] = models.RecommendQuests_Result{
			RecommendationService_questToAdd: models.RecommendationService_questToAdd{
				ID:          int(r.Id),
				Title:       r.Title,
				Description: r.Description,
				Category:    r.Category,
			},
			SimilarityScore: r.SimilarityScore,
			Explanation:     r.Explanation,
		}
	}

	profileInfo := make(map[string]any)
	if grpcResp.UserProfileInfo != nil {
		profileInfo["quests_count"] = grpcResp.UserProfileInfo.QuestsCount
		profileInfo["embedding_dim"] = grpcResp.UserProfileInfo.EmbeddingDim
		profileInfo["method"] = grpcResp.UserProfileInfo.Method
	}

	return &models.RecommendationService_RecommendQuests_Resp{
		Recommendations: recommendations,
		UserProfileInfo: profileInfo,
	}, nil
}
