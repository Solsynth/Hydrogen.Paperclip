package models

import "gorm.io/datatypes"

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

	Metadata datatypes.JSONMap `json:"metadata"`
	IsMature bool              `json:"is_mature"`

	Ref   *Attachment `json:"ref"`
	RefID *uint       `json:"ref_id"`

	Account   Account `json:"account"`
	AccountID uint    `json:"account_id"`
}
