package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChatGroup — сущность группового чата (название, аватар, владелец).
// group_uuid используется как внешний идентификатор в API и на фронте.
type ChatGroup struct {
	gorm.Model     `c_migrator:"enabled" c_migrator_t_name:"chat_groups"`
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

func (ChatGroup) TableName() string { return "chat_groups" }

var ChatGroupNotFoundErr = errors.New("chat group not found")

func CreateChatGroup(ctx context.Context, db *gorm.DB, g *ChatGroup) error {
	if g.GroupUuid == uuid.Nil {
		g.GroupUuid = uuid.New()
	}
	return db.WithContext(ctx).Create(g).Error
}

func GetChatGroupById(ctx context.Context, db *gorm.DB, id int) (ChatGroup, error) {
	var g ChatGroup
	result := db.WithContext(ctx).First(&g, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return g, ChatGroupNotFoundErr
	}
	return g, result.Error
}

func GetChatGroupByUuid(ctx context.Context, db *gorm.DB, groupUuid uuid.UUID) (ChatGroup, error) {
	var g ChatGroup
	result := db.WithContext(ctx).Where("group_uuid = ?", groupUuid).First(&g)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return g, ChatGroupNotFoundErr
	}
	return g, result.Error
}

func UpdateChatGroup(ctx context.Context, db *gorm.DB, g *ChatGroup) error {
	return db.WithContext(ctx).Save(g).Error
}

func ListChatGroupsForUser(ctx context.Context, db *gorm.DB, userId int) ([]ChatGroup, error) {
	var groups []ChatGroup
	err := db.WithContext(ctx).
		Table("chat_groups AS cg").
		Select("cg.*").
		Joins("JOIN chat_group_members AS m ON m.group_id = cg.id AND m.deleted_at IS NULL").
		Where("m.user_id = ? AND cg.deleted_at IS NULL", userId).
		Order("cg.updated_at DESC").
		Scan(&groups).Error
	return groups, err
}
