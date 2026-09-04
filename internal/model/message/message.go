package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"context"
	"errors"

	"gorm.io/gorm"
)

type Message struct {
	gorm.Model `c_migrator:"enabled"`
	Id         int    `json:"id" gorm:"type:int;unique;primaryKey;autoIncrement"`
	Text       string `json:"text" gorm:"type:text;not null"`
	CreatedAt  time.TimestampWithTimeZoneMicro
	UpdatedAt  time.TimestampWithTimeZoneMicro
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

var MessageNotFoundErr = errors.New("message not found")

func CreateMessage(ctx context.Context, db *gorm.DB, m *Message) error {
	return db.WithContext(ctx).Create(m).Error
}

func UpdateMessageById(ctx context.Context, db *gorm.DB, m *Message, id int) error {
	var message Message
	tx := db.WithContext(ctx)

	result := tx.First(&message, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return MessageNotFoundErr
	}

	return tx.Save(m).Error
}

func GetMessages(ctx context.Context, db *gorm.DB) ([]Message, error) {
	var messages []Message
	result := db.WithContext(ctx).Find(&messages)

	return messages, result.Error
}

func GetMessageById(ctx context.Context, db *gorm.DB, id int) (Message, error) {
	var message Message
	result := db.WithContext(ctx).First(&message, id)

	return message, result.Error
}

func DeleteMessageById(ctx context.Context, db *gorm.DB, id int) error {
	var message Message
	tx := db.WithContext(ctx)

	result := tx.First(&message, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return MessageNotFoundErr
	}

	return tx.Delete(&Message{}, id).Error
}
