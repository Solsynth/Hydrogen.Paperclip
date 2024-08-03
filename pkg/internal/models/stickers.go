package models

type Sticker struct {
	BaseModel

	Alias        string      `json:"alias"`
	Name         string      `json:"name"`
	AttachmentID uint        `json:"attachment_id"`
	Attachment   Attachment  `json:"attachment"`
	PackID       uint        `json:"pack_id"`
	Pack         StickerPack `json:"pack"`
	AccountID    uint        `json:"account_id"`
	Account      Account     `json:"account"`
}

type StickerPack struct {
	BaseModel

	Prefix      string    `json:"prefix"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Stickers    []Sticker `json:"stickers" gorm:"foreignKey:PackID;constraint:OnDelete:CASCADE"`
	AccountID   uint      `json:"account_id"`
	Account     Account   `json:"account"`
}
