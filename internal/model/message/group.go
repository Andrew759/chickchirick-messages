package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Group struct {
	gorm.Model `c_migrator:"enabled"`
	MessageId  int       `json:"message_id" gorm:"type:int"`
	Message    Message   `json:"message" gorm:"references:MessageId"`
	GroupId    uuid.UUID `json:"GroupId" gorm:"type:uuid"`
	CreatedAt  time.TimestampWithTimeZoneMicro
	UpdatedAt  time.TimestampWithTimeZoneMicro
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

var GroupNotFoundErr = errors.New("group not found")

func CreateGroup(db *gorm.DB, g *Group) error {
	return db.Create(g).Error
}

func UpdateGroupById(db *gorm.DB, g *Group, id int) error {
	var group Group
	result := db.First(&group, id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return GroupNotFoundErr
	}

	return db.Save(g).Error
}

func GetGroups(db *gorm.DB) ([]Group, error) {
	var groups []Group
	result := db.Find(&groups)

	return groups, result.Error
}

func GetGroupById(db *gorm.DB, id int) (Group, error) {
	var group Group
	result := db.First(&group, id)

	return group, result.Error
}

func DeleteGroupById(db *gorm.DB, id int) error {
	var group Group
	result := db.First(&group, id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return GroupNotFoundErr
	}

	return db.Delete(&Group{}, id).Error
}
