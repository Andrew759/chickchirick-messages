package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Channel — сущность канала (broadcast + E2EE для приватных).
// Публичные каналы могут быть без E2EE на клиенте; приватные — с ротацией ключа
// при изменении состава (как GroupMembersChanged).
type Channel struct {
	gorm.Model     `c_migrator:"enabled" c_migrator_t_name:"channels"`
	Id             int        `json:"id" gorm:"type:int;unique;primaryKey;autoIncrement"`
	ChannelUuid    uuid.UUID  `json:"channel_uuid" gorm:"type:uuid;not null;uniqueIndex"`
	Title          string     `json:"title" gorm:"type:varchar(255);not null"`
	Description    string     `json:"description" gorm:"type:text"`
	AvatarFileUuid *uuid.UUID `json:"avatar_file_uuid,omitempty" gorm:"type:uuid;index"`
	IsPublic       bool       `json:"is_public" gorm:"type:boolean;not null;default:false;index"`
	// E2EEEnabled — клиент обязан шифровать payload; сервер хранит ciphertext as-is.
	E2EEEnabled bool         `json:"e2ee_enabled" gorm:"column:e2ee_enabled;type:boolean;not null;default:true"`
	CreatedBy   int          `json:"created_by" gorm:"type:int;not null;index"`
	Creator     UserRelation `json:"creator,omitempty" gorm:"foreignKey:CreatedBy;references:UserId"`
	CreatedAt   time.TimestampWithTimeZoneMicro
	UpdatedAt   time.TimestampWithTimeZoneMicro
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (Channel) TableName() string { return "channels" }

var ChannelNotFoundErr = errors.New("channel not found")
var ChannelForbiddenErr = errors.New("not a channel member")
var ChannelPostForbiddenErr = errors.New("only owner/admin can post to channel")

func CreateChannel(ctx context.Context, db *gorm.DB, ch *Channel) error {
	if ch.ChannelUuid == uuid.Nil {
		ch.ChannelUuid = uuid.New()
	}
	return db.WithContext(ctx).Create(ch).Error
}

func GetChannelByUuid(ctx context.Context, db *gorm.DB, channelUuid uuid.UUID) (Channel, error) {
	var ch Channel
	result := db.WithContext(ctx).Where("channel_uuid = ?", channelUuid).First(&ch)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return ch, ChannelNotFoundErr
	}
	return ch, result.Error
}

func GetChannelById(ctx context.Context, db *gorm.DB, id int) (Channel, error) {
	var ch Channel
	result := db.WithContext(ctx).First(&ch, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return ch, ChannelNotFoundErr
	}
	return ch, result.Error
}

func UpdateChannel(ctx context.Context, db *gorm.DB, ch *Channel) error {
	return db.WithContext(ctx).Save(ch).Error
}

func ListChannelsForUser(ctx context.Context, db *gorm.DB, userId int) ([]Channel, error) {
	var channels []Channel
	err := db.WithContext(ctx).
		Table("channels AS c").
		Select("c.*").
		Joins("JOIN channel_members AS m ON m.channel_id = c.id AND m.deleted_at IS NULL").
		Where("m.user_id = ? AND c.deleted_at IS NULL", userId).
		Order("c.updated_at DESC").
		Scan(&channels).Error
	return channels, err
}

// ListPublicChannels — каталог публичных каналов (поиск / discovery).
func ListPublicChannels(ctx context.Context, db *gorm.DB, limit, offset int, query string) ([]Channel, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	q := db.WithContext(ctx).
		Where("is_public = ? AND deleted_at IS NULL", true)
	if query != "" {
		like := "%" + query + "%"
		q = q.Where("title ILIKE ? OR description ILIKE ?", like, like)
	}
	var channels []Channel
	err := q.Order("updated_at DESC").Limit(limit).Offset(offset).Find(&channels).Error
	return channels, err
}

func DeleteChannel(ctx context.Context, db *gorm.DB, id int) error {
	result := db.WithContext(ctx).Delete(&Channel{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ChannelNotFoundErr
	}
	return nil
}
