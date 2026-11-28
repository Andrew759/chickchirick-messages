package message

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChannelMedia struct {
	gorm.Model
	Id        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	ChannelId int       `json:"channel_id" gorm:"not null;index;uniqueIndex:idx_channel_media_unique"`
	FileUuid  uuid.UUID `json:"file_uuid" gorm:"type:uuid;not null;uniqueIndex:idx_channel_media_unique"`
	FileName  string    `json:"file_name" gorm:"type:varchar(255);not null;default:''"`
	Kind      string    `json:"kind" gorm:"type:varchar(20);not null;default:'audio'"`
}

func (ChannelMedia) TableName() string { return "channel_media" }

func ListChannelMedia(ctx context.Context, db *gorm.DB, channelId int) ([]ChannelMedia, error) {
	var rows []ChannelMedia
	err := db.WithContext(ctx).Where("channel_id = ? AND deleted_at IS NULL", channelId).Order("id ASC").Find(&rows).Error
	return rows, err
}

func AddChannelMedia(ctx context.Context, db *gorm.DB, row *ChannelMedia) error {
	if row.FileUuid == uuid.Nil {
		return errors.New("file_uuid is required")
	}
	if row.Kind == "" {
		row.Kind = "audio"
	}
	return db.WithContext(ctx).Where("channel_id = ? AND file_uuid = ?", row.ChannelId, row.FileUuid).FirstOrCreate(row).Error
}

func RemoveChannelMedia(ctx context.Context, db *gorm.DB, channelId int, fileUuid uuid.UUID) error {
	result := db.WithContext(ctx).Where("channel_id = ? AND file_uuid = ?", channelId, fileUuid).Delete(&ChannelMedia{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
