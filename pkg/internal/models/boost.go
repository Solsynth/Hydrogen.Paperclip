package models

import "git.solsynth.dev/hypernet/nexus/pkg/nex/cruda"

const (
	BoostStatusPending = iota
	BoostStatusActive
	BoostStatusSuspended
	BoostStatusError
)

// AttachmentBoost is made for speed up attachment loading by copy the original attachments
// to others faster CDN or storage destinations.
type AttachmentBoost struct {
	cruda.BaseModel

	Status      int `json:"status"`
	Destination int `json:"destination"`

	AttachmentID uint       `json:"attachment_id"`
	Attachment   Attachment `json:"attachment"`

	AccountID uint `json:"account"`
}
