package models

import (
	"git.solsynth.dev/hypernet/nexus/pkg/nex/cruda"
	"time"

	"gorm.io/datatypes"
)

type AttachmentDst = int8

const (
	AttachmentDstTemporary = AttachmentDst(iota)
	AttachmentDstPermanent
)

type Attachment struct {
	cruda.BaseModel

	// Random ID is for accessing (appear in URL)
	Rid string `json:"rid" gorm:"uniqueIndex"`
	// Unique ID is for storing (appear in local file name or object name)
	Uuid string `json:"uuid"`

	Size        int64         `json:"size"`
	Name        string        `json:"name"`
	Alternative string        `json:"alt"`
	MimeType    string        `json:"mimetype"`
	HashCode    string        `json:"hash"`
	Destination AttachmentDst `json:"destination"`
	RefCount    int           `json:"ref_count"`

	FileChunks datatypes.JSONMap `json:"file_chunks"`

	CleanedAt *time.Time `json:"cleaned_at"`

	Metadata   datatypes.JSONMap `json:"metadata"`
	IsMature   bool              `json:"is_mature"`
	IsAnalyzed bool              `json:"is_analyzed"`
	IsUploaded bool              `json:"is_uploaded"`
	IsSelfRef  bool              `json:"is_self_ref"`

	Ref   *Attachment `json:"ref"`
	RefID *uint       `json:"ref_id"`

	Pool   *AttachmentPool `json:"pool"`
	PoolID *uint           `json:"pool_id"`

	Account   Account `json:"account"`
	AccountID uint    `json:"account_id"`
}
