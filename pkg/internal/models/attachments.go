package models

import (
	"time"

	"git.solsynth.dev/hypernet/nexus/pkg/nex/cruda"

	"gorm.io/datatypes"
)

const (
	AttachmentDstTemporary = 0 // The destination 0 is a reserved config for pre-upload processing
)

type Attachment struct {
	cruda.BaseModel

	// Random ID is for accessing (appear in URL)
	Rid string `json:"rid" gorm:"uniqueIndex"`
	// Unique ID is for storing (appear in local file name or object name)
	Uuid string `json:"uuid"`

	Size        int64  `json:"size"`
	Name        string `json:"name"`
	Alternative string `json:"alt"`
	MimeType    string `json:"mimetype"`
	HashCode    string `json:"hash"`
	Destination int    `json:"destination"`
	RefCount    int    `json:"ref_count"`

	FileChunks datatypes.JSONMap `json:"file_chunks"`

	CleanedAt *time.Time `json:"cleaned_at"`

	Metadata datatypes.JSONMap `json:"metadata"` // This field is analyzer auto generated metadata
	Usermeta datatypes.JSONMap `json:"usermeta"` // This field is user set metadata

	Thumbnail     string `json:"thumbnail"`      // The cover image of audio / video attachment
	ContentRating int    `json:"content_rating"` // This field use to filter mature content or not
	QualityRating int    `json:"quality_rating"` // This field use to filter good content or not

	IsAnalyzed  bool `json:"is_analyzed"`
	IsUploaded  bool `json:"is_uploaded"`
	IsSelfRef   bool `json:"is_self_ref"`
	IsIndexable bool `json:"is_indexable"` // Show this attachment in the public directory api or not

	Ref   *Attachment `json:"ref"`
	RefID *uint       `json:"ref_id"`

	Pool   *AttachmentPool `json:"pool"`
	PoolID *uint           `json:"pool_id"`

	AccountID uint `json:"account_id"`

	// Outdated fields, just for backward compatibility
	IsMature bool `json:"is_mature"`
}
