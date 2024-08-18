package models

import (
	"gorm.io/datatypes"
	"time"
)

type AttachmentDst = int8

const (
	AttachmentDstTemporary = AttachmentDst(iota)
	AttachmentDstPermanent
)

type Attachment struct {
	BaseModel

	Uuid        string        `json:"uuid"`
	Size        int64         `json:"size"`
	Name        string        `json:"name"`
	Alternative string        `json:"alt"`
	Usage       string        `json:"usage"`
	MimeType    string        `json:"mimetype"`
	HashCode    string        `json:"hash"`
	Destination AttachmentDst `json:"destination"`
	RefCount    int           `json:"ref_count"`

	CleanedAt *time.Time `json:"cleaned_at"`

	Metadata   datatypes.JSONMap `json:"metadata"`
	IsMature   bool              `json:"is_mature"`
	IsAnalyzed bool              `json:"is_analyzed"`
	IsSelfRef  bool              `json:"is_self_ref"`

	Ref   *Attachment `json:"ref"`
	RefID *uint       `json:"ref_id"`

	Pool   *AttachmentPool `json:"pool"`
	PoolID *uint           `json:"pool_id"`

	Account   Account `json:"account"`
	AccountID uint    `json:"account_id"`
}
