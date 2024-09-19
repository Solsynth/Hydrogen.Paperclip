package grpc

import (
	"context"
	"strconv"

	"git.solsynth.dev/hydrogen/dealer/pkg/proto"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/database"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/models"
)

func (v *Server) BroadcastDeletion(ctx context.Context, request *proto.DeletionRequest) (*proto.DeletionResponse, error) {
	switch request.GetResourceType() {
	case "account":
		numericId, err := strconv.Atoi(request.GetResourceId())
		if err != nil {
			break
		}
		for _, model := range database.AutoMaintainRange {
			switch model.(type) {
			default:
				database.C.Delete(model, "account_id = ?", numericId)
			}
		}
		database.C.Delete(&models.Account{}, "id = ?", numericId)
	}

	return &proto.DeletionResponse{}, nil
}
