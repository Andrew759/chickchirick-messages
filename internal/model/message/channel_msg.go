package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChannelMsg — связь message → channel (аналог Group для групповых чатов).
// Text в Message хранит ciphertext при e2ee_enabled=true на канале.
type ChannelMsg struct {
	gorm.Model `c_migrator:"enabled" c_migrator_t_name:"channel_messages"`
	Id         int          `json:"id" gorm:"type:int;unique;primaryKey;autoIncrement"`
	MessageId  int          `json:"message_id" gorm:"type:int;not null;index"`
	Message    Message      `json:"message" gorm:"foreignKey:MessageId"`
	ChannelId  uuid.UUID    `json:"channel_id" gorm:"type:uuid;not null;index"`
	SenderId   int          `json:"sender_id" gorm:"type:int;not null;index;default:0"`
	Sender     UserRelation `json:"sender,omitempty" gorm:"foreignKey:SenderId;references:UserId"`
	CreatedAt  time.TimestampWithTimeZoneMicro
	UpdatedAt  time.TimestampWithTimeZoneMicro
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (ChannelMsg) TableName() string { return "channel_messages" }

var ChannelMsgNotFoundErr = errors.New("channel message not found")

func CreateChannelMsg(ctx context.Context, db *gorm.DB, cm *ChannelMsg) error {
	return db.WithContext(ctx).Create(cm).Error
}

func GetChannelMsgByMessageId(ctx context.Context, db *gorm.DB, messageId int) (ChannelMsg, error) {
	var cm ChannelMsg
	result := db.WithContext(ctx).Where("message_id = ?", messageId).First(&cm)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return cm, ChannelMsgNotFoundErr
	}
	return cm, result.Error
}

func GetChannelMsgsByChannelId(ctx context.Context, db *gorm.DB, channelId uuid.UUID, limit, offset int) ([]ChannelMsg, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rows []ChannelMsg
	err := db.WithContext(ctx).
		Where("channel_id = ? AND deleted_at IS NULL", channelId).
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error
	return rows, err
}

func DeleteChannelMsgById(ctx context.Context, db *gorm.DB, id int) error {
	result := db.WithContext(ctx).Delete(&ChannelMsg{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ChannelMsgNotFoundErr
	}
	return nil
}
