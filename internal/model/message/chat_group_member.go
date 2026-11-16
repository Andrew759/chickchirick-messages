package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"context"
	"errors"

	"gorm.io/gorm"
)

const (
	GroupRoleOwner  = "owner"
	GroupRoleAdmin  = "admin"
	GroupRoleMember = "member"
)

// ChatGroupMember — участник группового чата.
type ChatGroupMember struct {
	gorm.Model `c_migrator:"enabled" c_migrator_t_name:"chat_group_members"`
	Id         int          `json:"id" gorm:"type:int;unique;primaryKey;autoIncrement"`
	GroupId    int          `json:"group_id" gorm:"type:int;not null;index;uniqueIndex:idx_group_user"`
	Group      ChatGroup    `json:"group,omitempty" gorm:"foreignKey:GroupId"`
	UserId     int          `json:"user_id" gorm:"type:int;not null;index;uniqueIndex:idx_group_user"`
	User       UserRelation `json:"user,omitempty" gorm:"foreignKey:UserId;references:UserId"`
	Role       string       `json:"role" gorm:"type:varchar(20);not null;default:'member'"`
	CreatedAt  time.TimestampWithTimeZoneMicro
	UpdatedAt  time.TimestampWithTimeZoneMicro
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (ChatGroupMember) TableName() string { return "chat_group_members" }

var ChatGroupMemberNotFoundErr = errors.New("chat group member not found")
var ChatGroupForbiddenErr = errors.New("not a group member")

func CreateChatGroupMember(ctx context.Context, db *gorm.DB, m *ChatGroupMember) error {
	if m.Role == "" {
		m.Role = GroupRoleMember
	}
	return db.WithContext(ctx).Create(m).Error
}

func GetGroupMember(ctx context.Context, db *gorm.DB, groupId, userId int) (ChatGroupMember, error) {
	var m ChatGroupMember
	result := db.WithContext(ctx).
		Where("group_id = ? AND user_id = ?", groupId, userId).
		First(&m)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return m, ChatGroupMemberNotFoundErr
	}
	return m, result.Error
}

func IsGroupMember(ctx context.Context, db *gorm.DB, groupId, userId int) (bool, error) {
	_, err := GetGroupMember(ctx, db, groupId, userId)
	if errors.Is(err, ChatGroupMemberNotFoundErr) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func ListGroupMembers(ctx context.Context, db *gorm.DB, groupId int) ([]ChatGroupMember, error) {
	var members []ChatGroupMember
	err := db.WithContext(ctx).Where("group_id = ?", groupId).Find(&members).Error
	return members, err
}

func ListGroupMemberUserIds(ctx context.Context, db *gorm.DB, groupId int) ([]int, error) {
	var ids []int
	err := db.WithContext(ctx).
		Model(&ChatGroupMember{}).
		Where("group_id = ?", groupId).
		Pluck("user_id", &ids).Error
	return ids, err
}

func DeleteGroupMember(ctx context.Context, db *gorm.DB, groupId, userId int) error {
	result := db.WithContext(ctx).
		Where("group_id = ? AND user_id = ?", groupId, userId).
		Delete(&ChatGroupMember{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ChatGroupMemberNotFoundErr
	}
	return nil
}
