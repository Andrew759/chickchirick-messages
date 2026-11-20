package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"context"
	"errors"

	"gorm.io/gorm"
)

const (
	ChannelRoleOwner      = "owner"
	ChannelRoleAdmin      = "admin"
	ChannelRoleSubscriber = "subscriber"
)

// ChannelMember — подписчик / администратор канала.
type ChannelMember struct {
	gorm.Model `c_migrator:"enabled" c_migrator_t_name:"channel_members"`
	Id         int          `json:"id" gorm:"type:int;unique;primaryKey;autoIncrement"`
	ChannelId  int          `json:"channel_id" gorm:"type:int;not null;index;uniqueIndex:idx_channel_user"`
	Channel    Channel      `json:"channel,omitempty" gorm:"foreignKey:ChannelId"`
	UserId     int          `json:"user_id" gorm:"type:int;not null;index;uniqueIndex:idx_channel_user"`
	User       UserRelation `json:"user,omitempty" gorm:"foreignKey:UserId;references:UserId"`
	Role       string       `json:"role" gorm:"type:varchar(20);not null;default:'subscriber'"`
	CreatedAt  time.TimestampWithTimeZoneMicro
	UpdatedAt  time.TimestampWithTimeZoneMicro
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (ChannelMember) TableName() string { return "channel_members" }

var ChannelMemberNotFoundErr = errors.New("channel member not found")

func CreateChannelMember(ctx context.Context, db *gorm.DB, m *ChannelMember) error {
	if m.Role == "" {
		m.Role = ChannelRoleSubscriber
	}
	return db.WithContext(ctx).Create(m).Error
}

func GetChannelMember(ctx context.Context, db *gorm.DB, channelId, userId int) (ChannelMember, error) {
	var m ChannelMember
	result := db.WithContext(ctx).
		Where("channel_id = ? AND user_id = ?", channelId, userId).
		First(&m)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return m, ChannelMemberNotFoundErr
	}
	return m, result.Error
}

func IsChannelMember(ctx context.Context, db *gorm.DB, channelId, userId int) (bool, error) {
	_, err := GetChannelMember(ctx, db, channelId, userId)
	if errors.Is(err, ChannelMemberNotFoundErr) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func CanPostToChannel(ctx context.Context, db *gorm.DB, channelId, userId int) (bool, error) {
	m, err := GetChannelMember(ctx, db, channelId, userId)
	if err != nil {
		if errors.Is(err, ChannelMemberNotFoundErr) {
			return false, nil
		}
		return false, err
	}
	return m.Role == ChannelRoleOwner || m.Role == ChannelRoleAdmin, nil
}

func ListChannelMembers(ctx context.Context, db *gorm.DB, channelId int) ([]ChannelMember, error) {
	var members []ChannelMember
	err := db.WithContext(ctx).Where("channel_id = ?", channelId).Find(&members).Error
	return members, err
}

func ListChannelMemberUserIds(ctx context.Context, db *gorm.DB, channelId int) ([]int, error) {
	var ids []int
	err := db.WithContext(ctx).
		Model(&ChannelMember{}).
		Where("channel_id = ?", channelId).
		Pluck("user_id", &ids).Error
	return ids, err
}

func CountChannelMembers(ctx context.Context, db *gorm.DB, channelId int) (int64, error) {
	var n int64
	err := db.WithContext(ctx).
		Model(&ChannelMember{}).
		Where("channel_id = ? AND deleted_at IS NULL", channelId).
		Count(&n).Error
	return n, err
}

func UpdateChannelMemberRole(ctx context.Context, db *gorm.DB, channelId, userId int, role string) error {
	result := db.WithContext(ctx).
		Model(&ChannelMember{}).
		Where("channel_id = ? AND user_id = ?", channelId, userId).
		Update("role", role)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ChannelMemberNotFoundErr
	}
	return nil
}

func DeleteChannelMember(ctx context.Context, db *gorm.DB, channelId, userId int) error {
	result := db.WithContext(ctx).
		Where("channel_id = ? AND user_id = ?", channelId, userId).
		Delete(&ChannelMember{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ChannelMemberNotFoundErr
	}
	return nil
}
