package grpc

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/emptypb"

	"git.solsynth.dev/hydrogen/paperclip/pkg/database"
	"git.solsynth.dev/hydrogen/paperclip/pkg/grpc/proto"
	"git.solsynth.dev/hydrogen/paperclip/pkg/models"
	jsoniter "github.com/json-iterator/go"
)

func (v *Server) GetAttachment(ctx context.Context, request *proto.AttachmentLookupRequest) (*proto.Attachment, error) {
	var attachment models.Attachment

	tx := database.C.Model(&models.Attachment{})
	if request.Id != nil {
		tx = tx.Where("id = ?", request.GetId())
	}
	if request.Uuid != nil {
		tx = tx.Where("uuid = ?", request.GetUuid())
	}
	if request.Usage != nil {
		tx = tx.Where("usage = ?", request.GetUsage())
	}

	if err := tx.First(&attachment).Error; err != nil {
		return nil, err
	}

	rawMetadata, _ := jsoniter.Marshal(attachment.Metadata)

	return &proto.Attachment{
		Id:          uint64(attachment.ID),
		Uuid:        attachment.Uuid,
		Size:        attachment.Size,
		Name:        attachment.Name,
		Alt:         attachment.Alternative,
		Usage:       attachment.Usage,
		Mimetype:    attachment.MimeType,
		Hash:        attachment.HashCode,
		Destination: attachment.Destination,
		Metadata:    rawMetadata,
		IsMature:    attachment.IsMature,
		AccountId:   uint64(attachment.AccountID),
	}, nil
}

func (v *Server) CheckAttachmentExists(ctx context.Context, request *proto.AttachmentLookupRequest) (*emptypb.Empty, error) {
	tx := database.C.Model(&models.Attachment{})
	if request.Id != nil {
		tx = tx.Where("id = ?", request.GetId())
	}
	if request.Uuid != nil {
		tx = tx.Where("uuid = ?", request.GetUuid())
	}
	if request.Usage != nil {
		tx = tx.Where("usage = ?", request.GetUsage())
	}

	var count int64
	if err := tx.Model(&models.Attachment{}).Count(&count).Error; err != nil {
		return nil, err
	} else if count == 0 {
		return nil, fmt.Errorf("record not found")
	}

	return &emptypb.Empty{}, nil
}
