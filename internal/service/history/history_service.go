package history

import (
	"context"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"
)

type Message struct {
	ID               int    `json:"id"`
	SenderID         int    `json:"senderId"`
	RecipientID      int    `json:"recipientId,omitempty"`
	GroupID          string `json:"groupId,omitempty"`
	ChannelID        string `json:"channelId,omitempty"`
	Text             string `json:"text"`
	CreatedAt        string `json:"createdAt"`
	RespondMessageID *int   `json:"respondMessageId,omitempty"`
}

type History struct {
	UserID   int       `json:"userId"`
	Messages []Message `json:"messages"`
}

type row struct {
	ID               int
	SenderID         int
	RecipientID      int
	GroupID          *string
	ChannelID        *string
	Text             string
	CreatedAt        time.Time
	RespondMessageID *int
}

// GetHistory returns personal + group messages for the authenticated user.
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

	var personalRows []row
	err := db.WithContext(ctx).
		Table("personals AS p").
		Select("m.id, p.sender_id AS sender_id, p.recipient_id AS recipient_id, NULL::text AS group_id, m.text, m.created_at, message_meta.respond_message_id").
		Joins("JOIN messages AS m ON m.id = p.message_id").
		Joins("LEFT JOIN message_meta ON message_meta.message_id = m.id").
		Where("(p.sender_id = ? OR p.recipient_id = ?) AND m.deleted_at IS NULL AND p.deleted_at IS NULL", userID, userID).
		Order("m.created_at ASC, m.id ASC").
		Scan(&personalRows).Error
	if err != nil {
		return History{}, err
	}

	var groupRows []row
	err = db.WithContext(ctx).
		Table("group_messages AS g").
		Select("m.id, g.sender_id AS sender_id, NULL::int AS recipient_id, g.group_id::text AS group_id, m.text, m.created_at, message_meta.respond_message_id").
		Joins("JOIN messages AS m ON m.id = g.message_id").
		Joins("LEFT JOIN message_meta ON message_meta.message_id = m.id").
		Joins("JOIN group_meta AS cg ON cg.group_uuid = g.group_id AND cg.deleted_at IS NULL").
		Joins("JOIN group_members AS cgm ON cgm.group_id = cg.id AND cgm.user_id = ? AND cgm.deleted_at IS NULL", userID).
		Where("m.deleted_at IS NULL AND g.deleted_at IS NULL").
		Order("m.created_at ASC, m.id ASC").
		Scan(&groupRows).Error
	if err != nil {
		groupRows = nil
	}

	var channelRows []row
	err = db.WithContext(ctx).
		Table("channel_messages AS cm").
		Select("m.id, cm.sender_id AS sender_id, NULL::int AS recipient_id, NULL::text AS group_id, cm.channel_id::text AS channel_id, m.text, m.created_at, message_meta.respond_message_id").
		Joins("JOIN messages AS m ON m.id = cm.message_id").
		Joins("LEFT JOIN message_meta ON message_meta.message_id = m.id").
		Joins("JOIN channels AS ch ON ch.channel_uuid = cm.channel_id AND ch.deleted_at IS NULL").
		Joins("JOIN channel_members AS cmem ON cmem.channel_id = ch.id AND cmem.user_id = ? AND cmem.deleted_at IS NULL", userID).
		Where("m.deleted_at IS NULL AND cm.deleted_at IS NULL").
		Order("m.created_at ASC, m.id ASC").
		Scan(&channelRows).Error
	if err != nil {
		channelRows = nil
	}

	messages := make([]Message, 0, len(personalRows)+len(groupRows)+len(channelRows))
	for _, r := range personalRows {
		messages = append(messages, Message{
			ID:               r.ID,
			SenderID:         r.SenderID,
			RecipientID:      r.RecipientID,
			Text:             r.Text,
			CreatedAt:        r.CreatedAt.UTC().Format(time.RFC3339Nano),
			RespondMessageID: r.RespondMessageID,
		})
	}
	for _, r := range groupRows {
		gid := ""
		if r.GroupID != nil {
			gid = *r.GroupID
		}
		messages = append(messages, Message{
			ID:               r.ID,
			SenderID:         r.SenderID,
			GroupID:          gid,
			Text:             r.Text,
			CreatedAt:        r.CreatedAt.UTC().Format(time.RFC3339Nano),
			RespondMessageID: r.RespondMessageID,
		})
	}
	for _, r := range channelRows {
		cid := ""
		if r.ChannelID != nil {
			cid = *r.ChannelID
		}
		messages = append(messages, Message{
			ID:               r.ID,
			SenderID:         r.SenderID,
			ChannelID:        cid,
			Text:             r.Text,
			CreatedAt:        r.CreatedAt.UTC().Format(time.RFC3339Nano),
			RespondMessageID: r.RespondMessageID,
		})
	}

	sort.SliceStable(messages, func(i, j int) bool {
		return messages[i].CreatedAt < messages[j].CreatedAt
	})

	return History{UserID: userID, Messages: messages}, nil
}
