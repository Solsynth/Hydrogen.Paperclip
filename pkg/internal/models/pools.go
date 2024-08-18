package models

import "gorm.io/datatypes"

type AttachmentPool struct {
	BaseModel

	Alias       string                                   `json:"alias"`
	Name        string                                   `json:"name"`
	Description string                                   `json:"description"`
	Config      datatypes.JSONType[AttachmentPoolConfig] `json:"config"`

	Attachments []Attachment `json:"attachments"`

	Account   *Account `json:"account"`
	AccountID *uint    `json:"account_id"`
}

type AttachmentPoolConfig struct {
	ExistLifecycle     *int `json:"exist_lifecycle"`
	IsPublicAccessible bool `json:"is_public_accessible"`
}
