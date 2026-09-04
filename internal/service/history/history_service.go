package history

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type Message struct {
	ID               int    `json:"id"`
	SenderID         int    `json:"senderId"`
	RecipientID      int    `json:"recipientId"`
	Text             string `json:"text"`
	CreatedAt        any    `json:"createdAt,omitempty"`
	RespondMessageID *int   `json:"respondMessageId,omitempty"`
}

type History struct {
	UserID   int       `json:"userId"`
	Messages []Message `json:"messages"`
}

// GetHistory returns messages belonging to the authenticated user.
func GetHistory(ctx context.Context, db *gorm.DB, userUUID string) (History, error) {
	var userID int
	if err := db.WithContext(ctx).
		Table("user_relations").
		Select("user_id").
		Where("user_uuid = ?", userUUID).
		Scan(&userID).Error; err != nil {
		return History{}, err
	}
	if userID == 0 {
		return History{}, fmt.Errorf("user relation not found for uuid %s", userUUID)
	}

	var rows []struct {
		ID               int
		SenderID         int
		RecipientID      int
		Text             string
		CreatedAt        any
		RespondMessageID *int
	}

	err := db.WithContext(ctx).
		Table("personals AS p").
		Select("m.id, p.sender_id, p.recipient_id, m.text, m.created_at, message_meta.respond_message_id").
		Joins("JOIN messages AS m ON m.id = p.message_id").
		Joins("LEFT JOIN message_meta ON message_meta.message_id = m.id").
		Where("p.sender_id = ? OR p.recipient_id = ?", userID, userID).
		Order("m.created_at ASC, m.id ASC").
		Scan(&rows).Error
	if err != nil {
		return History{}, err
	}

	messages := make([]Message, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, Message{
			ID:               row.ID,
			SenderID:         row.SenderID,
			RecipientID:      row.RecipientID,
			Text:             row.Text,
			CreatedAt:        row.CreatedAt,
			RespondMessageID: row.RespondMessageID,
		})
	}

	return History{UserID: userID, Messages: messages}, nil
}
