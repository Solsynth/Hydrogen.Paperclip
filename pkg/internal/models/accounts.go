package models

import "git.solsynth.dev/hypernet/nexus/pkg/nex/sec"

type Account struct {
	sec.UserInfo

	Attachments []Attachment     `json:"attachments"`
	Pools       []AttachmentPool `json:"pools"`
}
