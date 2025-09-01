package dto

import "time"

type MessageDTO struct {
	SenderID    string        `json:"sender_id"`
	ReceiverId  string        `json:"receiver_id"`
	MessageType string        `json:"message_type" validate:"required, oneof=text image file video audio"`
	Content     string        `json:"content"`
	Time        time.Duration `json:"time"`
	Status      string        `json:"status" validate:"required, oneof=send withdraw"`
}
