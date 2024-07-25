package models

import "gorm.io/datatypes"

type Attachment struct {
	BaseModel

	Uuid        string `json:"uuid"`
	Size        int64  `json:"size"`
	Name        string `json:"name"`
	Alternative string `json:"alt"`
	Usage       string `json:"usage"`
	MimeType    string `json:"mimetype"`
	HashCode    string `json:"hash"`
	Destination string `json:"destination"`

	Metadata datatypes.JSONMap `json:"metadata"`
	IsMature bool              `json:"is_mature"`

	Account   *Account `json:"account"`
	AccountID *uint    `json:"account_id"`
}
