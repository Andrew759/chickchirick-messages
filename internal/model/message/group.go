package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Group struct {
	gorm.Model `c_migrator:"enabled"`
	Id         int       `json:"id" gorm:"type:int;unique;primaryKey;autoIncrement"`
	MessageId  int       `json:"message_id" gorm:"type:int"`
	Message    Message   `json:"message" gorm:"foreignKey:MessageId"`
	GroupId    uuid.UUID `json:"GroupId" gorm:"type:uuid"`
	CreatedAt  time.TimestampWithTimeZoneMicro
	UpdatedAt  time.TimestampWithTimeZoneMicro
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

var GroupNotFoundErr = errors.New("group not found")

func CreateGroup(ctx context.Context, db *gorm.DB, g *Group) error {
	return db.WithContext(ctx).Create(g).Error
}

func UpdateGroupById(ctx context.Context, db *gorm.DB, g *Group, id int) error {
	var group Group
	tx := db.WithContext(ctx)

	result := tx.First(&group, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return GroupNotFoundErr
	}

	return tx.Save(g).Error
}

func GetGroups(ctx context.Context, db *gorm.DB) ([]Group, error) {
	var groups []Group
	result := db.WithContext(ctx).Find(&groups)

	return groups, result.Error
}

func GetGroupById(ctx context.Context, db *gorm.DB, id int) (Group, error) {
	var group Group
	result := db.WithContext(ctx).First(&group, id)

	return group, result.Error
}

func DeleteGroupById(ctx context.Context, db *gorm.DB, id int) error {
	var group Group
	tx := db.WithContext(ctx)

	result := tx.First(&group, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return GroupNotFoundErr
	}

	return tx.Delete(&Group{}, id).Error
}
