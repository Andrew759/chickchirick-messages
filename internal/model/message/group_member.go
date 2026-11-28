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

// GroupMember — участник группового чата.
type GroupMember struct {
	gorm.Model `c_migrator:"enabled" c_migrator_t_name:"chat_group_members"`
	Id         int          `json:"id" gorm:"type:int;unique;primaryKey;autoIncrement"`
	GroupId    int          `json:"group_id" gorm:"type:int;not null;index;uniqueIndex:idx_group_user"`
	Group      GroupMeta    `json:"group" gorm:"foreignKey:GroupId"`
	UserId     int          `json:"user_id" gorm:"type:int;not null;index;uniqueIndex:idx_group_user"`
	User       UserRelation `json:"user" gorm:"foreignKey:UserId;references:UserId"`
	Role       string       `json:"role" gorm:"type:varchar(20);not null;default:'member'"`
	CreatedAt  time.TimestampWithTimeZoneMicro
	UpdatedAt  time.TimestampWithTimeZoneMicro
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (GroupMember) TableName() string { return "group_members" }

var GroupMemberNotFoundErr = errors.New("group member not found")
var GroupForbiddenErr = errors.New("not a group member")

func CreateGroupMember(ctx context.Context, db *gorm.DB, m *GroupMember) error {
	return db.WithContext(ctx).Create(m).Error
}

func GetGroupMember(ctx context.Context, db *gorm.DB, groupId, userId int) (GroupMember, error) {
	var m GroupMember
	result := db.WithContext(ctx).
		Where("group_id = ? AND user_id = ?", groupId, userId).
		First(&m)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return m, GroupMemberNotFoundErr
	}
	return m, result.Error
}

func IsGroupMember(ctx context.Context, db *gorm.DB, groupId, userId int) (bool, error) {
	_, err := GetGroupMember(ctx, db, groupId, userId)
	if errors.Is(err, GroupMemberNotFoundErr) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func ListGroupMembers(ctx context.Context, db *gorm.DB, groupId int) ([]GroupMember, error) {
	var members []GroupMember
	err := db.WithContext(ctx).Where("group_id = ?", groupId).Find(&members).Error
	return members, err
}

func ListGroupMemberUserIds(ctx context.Context, db *gorm.DB, groupId int) ([]int, error) {
	var ids []int
	err := db.WithContext(ctx).
		Model(&GroupMember{}).
		Where("group_id = ?", groupId).
		Pluck("user_id", &ids).Error
	return ids, err
}

func DeleteGroupMember(ctx context.Context, db *gorm.DB, groupId, userId int) error {
	result := db.WithContext(ctx).
		Where("group_id = ? AND user_id = ?", groupId, userId).
		Delete(&GroupMember{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return GroupMemberNotFoundErr
	}
	return nil
}
