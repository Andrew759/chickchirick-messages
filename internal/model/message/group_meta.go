package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GroupMeta — сущность группового чата
type GroupMeta struct {
	gorm.Model     `c_migrator:"enabled" c_migrator_t_name:"gru"`
	Id             int          `json:"id" gorm:"type:int;unique;primaryKey;autoIncrement"`
	GroupUuid      uuid.UUID    `json:"group_uuid" gorm:"type:uuid;not null;uniqueIndex"`
	Title          string       `json:"title" gorm:"type:varchar(255);not null"`
	AvatarFileUuid *uuid.UUID   `json:"avatar_file_uuid,omitempty" gorm:"type:uuid;index"`
	CreatedBy      int          `json:"created_by" gorm:"type:int;not null;index"`
	Creator        UserRelation `json:"creator,omitempty" gorm:"foreignKey:CreatedBy;references:UserId"`
	CreatedAt      time.TimestampWithTimeZoneMicro
	UpdatedAt      time.TimestampWithTimeZoneMicro
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

func (GroupMeta) TableName() string { return "group_meta" }

var GroupMetaNotFoundErr = errors.New("group meta not found")

func CreateGroupMeta(ctx context.Context, db *gorm.DB, g *GroupMeta) error {
	return db.WithContext(ctx).Create(g).Error
}

func GetGroupMetaByUuid(ctx context.Context, db *gorm.DB, groupUuid uuid.UUID) (GroupMeta, error) {
	var g GroupMeta
	result := db.WithContext(ctx).Where("group_uuid = ?", groupUuid).First(&g)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return g, GroupMetaNotFoundErr
	}
	return g, result.Error
}

func UpdateChatGroup(ctx context.Context, db *gorm.DB, g *GroupMeta) error {
	return db.WithContext(ctx).Save(g).Error
}

func ListGroupMetasForUser(ctx context.Context, db *gorm.DB, userId int) ([]GroupMeta, error) {
	var groups []GroupMeta
	err := db.WithContext(ctx).
		Table("group_meta AS gm").
		Select("gm.*").
		Joins("JOIN group_members AS m ON m.group_id = gm.id AND m.deleted_at IS NULL").
		Where("m.user_id = ? AND gm.deleted_at IS NULL", userId).
		Order("gm.updated_at DESC").
		Scan(&groups).Error
	return groups, err
}

func DeleteGroupMeta(ctx context.Context, db *gorm.DB, id int) error {
	result := db.WithContext(ctx).Delete(&GroupMeta{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return GroupMetaNotFoundErr
	}
	return nil
}
