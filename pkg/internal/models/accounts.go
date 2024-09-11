package models

import "git.solsynth.dev/hydrogen/dealer/pkg/hyper"

type Account struct {
	hyper.BaseUser

	Attachments []Attachment     `json:"attachments"`
	Pools       []AttachmentPool `json:"pools"`
}
