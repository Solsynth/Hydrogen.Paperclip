package models

import (
	"git.solsynth.dev/hydrogen/dealer/pkg/hyper"
	"gorm.io/datatypes"
)

type AttachmentPool struct {
	hyper.BaseModel

	Alias       string                                   `json:"alias"`
	Name        string                                   `json:"name"`
	Description string                                   `json:"description"`
	Config      datatypes.JSONType[AttachmentPoolConfig] `json:"config"`

	Attachments []Attachment `json:"attachments" gorm:"foreignKey:PoolID"`

	Account   *Account `json:"account"`
	AccountID *uint    `json:"account_id"`
}

type AttachmentPoolConfig struct {
	MaxFileSize           *int64 `json:"max_file_size"`
	ExistLifecycle        *int64 `json:"exist_lifecycle"`
	AllowCrossPoolIngress bool   `json:"allow_cross_pool_ingress"`
	AllowCrossPoolEgress  bool   `json:"allow_cross_pool_egress"`
	PublicIndexable       bool   `json:"public_indexable"`
}
