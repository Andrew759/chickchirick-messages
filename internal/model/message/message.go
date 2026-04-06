package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Message struct {
	gorm.Model `c_migrator:"enabled"`
	Id         int                             `json:"id" gorm:"type:int;unique;primaryKey;autoIncrement"`
	Text       uuid.UUID                       `json:"text" gorm:"type:text;not null"`
	Date       time.TimestampWithTimeZoneMicro `json:"start_date" gorm:"type:timestamp without time zone"`
	CreatedAt  time.TimestampWithTimeZoneMicro
	UpdatedAt  time.TimestampWithTimeZoneMicro
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

var MessageNotFoundErr = errors.New("message not found")

func CreateMessage(db *gorm.DB, m *Message) error {
	return db.Create(m).Error
}

func UpdateMessageById(db *gorm.DB, m *Message, id int) error {
	var message Message
	result := db.First(&message, id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return MessageNotFoundErr
	}

	return db.Save(m).Error
}

func GetMessages(db *gorm.DB) ([]Message, error) {
	var messages []Message
	result := db.Find(&messages)

	return messages, result.Error
}

func GetMessageById(db *gorm.DB, id int) (Message, error) {
	var message Message
	result := db.First(&message, id)

	return message, result.Error
}

func DeleteMessageById(db *gorm.DB, id int) error {
	var message Message
	result := db.First(&message, id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return MessageNotFoundErr
	}

	return db.Delete(&Message{}, id).Error
}
