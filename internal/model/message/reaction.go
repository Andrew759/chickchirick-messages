package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"context"
	"errors"
	"unicode/utf8"

	"gorm.io/gorm"
)

// Reaction — реакция пользователя на сообщение (emoji).
type Reaction struct {
	gorm.Model `c_migrator:"enabled" c_migrator_t_name:"message_reactions"`
	Id         int          `json:"id" gorm:"type:int;unique;primaryKey;autoIncrement"`
	MessageId  int          `json:"message_id" gorm:"type:int;not null;index;uniqueIndex:idx_msg_user_emoji"`
	Message    Message      `json:"message,omitempty" gorm:"foreignKey:MessageId"`
	UserId     int          `json:"user_id" gorm:"type:int;not null;index;uniqueIndex:idx_msg_user_emoji"`
	User       UserRelation `json:"user,omitempty" gorm:"foreignKey:UserId;references:UserId"`
	Emoji      string       `json:"emoji" gorm:"type:varchar(32);not null;uniqueIndex:idx_msg_user_emoji"`
	CreatedAt  time.TimestampWithTimeZoneMicro
	UpdatedAt  time.TimestampWithTimeZoneMicro
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (Reaction) TableName() string { return "message_reactions" }

var ReactionNotFoundErr = errors.New("reaction not found")
var ReactionInvalidEmojiErr = errors.New("invalid emoji")

func validateEmoji(emoji string) error {
	if emoji == "" || utf8.RuneCountInString(emoji) > 8 {
		return ReactionInvalidEmojiErr
	}
	return nil
}

func CreateReaction(ctx context.Context, db *gorm.DB, r *Reaction) error {
	if err := validateEmoji(r.Emoji); err != nil {
		return err
	}
	return db.WithContext(ctx).Create(r).Error
}

func DeleteReaction(ctx context.Context, db *gorm.DB, messageID, userID int, emoji string) error {
	// hard delete — иначе uniqueIndex (message_id,user_id,emoji) блокирует повторный toggle
	result := db.WithContext(ctx).Unscoped().
		Where("message_id = ? AND user_id = ? AND emoji = ?", messageID, userID, emoji).
		Delete(&Reaction{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ReactionNotFoundErr
	}
	return nil
}

func GetReactionsByMessageId(ctx context.Context, db *gorm.DB, messageID int) ([]Reaction, error) {
	var rows []Reaction
	err := db.WithContext(ctx).
		Where("message_id = ? AND deleted_at IS NULL", messageID).
		Order("id ASC").
		Find(&rows).Error
	return rows, err
}

func GetReactionsByMessageIds(ctx context.Context, db *gorm.DB, messageIDs []int) ([]Reaction, error) {
	if len(messageIDs) == 0 {
		return nil, nil
	}
	var rows []Reaction
	err := db.WithContext(ctx).
		Where("message_id IN ? AND deleted_at IS NULL", messageIDs).
		Order("message_id ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

// FindReaction — есть ли конкретная реакция пользователя.
func FindReaction(ctx context.Context, db *gorm.DB, messageID, userID int, emoji string) (Reaction, error) {
	var r Reaction
	result := db.WithContext(ctx).
		Where("message_id = ? AND user_id = ? AND emoji = ? AND deleted_at IS NULL", messageID, userID, emoji).
		First(&r)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return r, ReactionNotFoundErr
	}
	return r, result.Error
}
