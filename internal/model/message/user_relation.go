package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRelation struct {
	gorm.Model `c_migrator:"enabled"`
	UserId     int       `json:"id" gorm:"type:int;unique;primaryKey;autoIncrement"`
	UserUuid   uuid.UUID `json:"user_uuid" gorm:"type:uuid"`
	CreatedAt  time.TimestampWithTimeZoneMicro
	UpdatedAt  time.TimestampWithTimeZoneMicro
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

var UserRelationNotFoundErr = errors.New("user relation not found")

func CreateUserRelation(db *gorm.DB, ur *UserRelation) error {
	return db.Create(ur).Error
}

func UpdateUserRelationById(db *gorm.DB, ur *UserRelation, id int) error {
	var userRelation UserRelation
	result := db.First(&userRelation, id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return UserRelationNotFoundErr
	}

	return db.Save(ur).Error
}

func GetUserRelation(db *gorm.DB) ([]UserRelation, error) {
	var userRelation []UserRelation
	result := db.Find(&userRelation)

	return userRelation, result.Error
}

func GetUserRelationById(db *gorm.DB, id int) (UserRelation, error) {
	var userRelation UserRelation
	result := db.First(&userRelation, id)

	return userRelation, result.Error
}

func DeleteUserRelationById(db *gorm.DB, id int) error {
	var userRelation UserRelation
	result := db.First(&userRelation, id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return UserRelationNotFoundErr
	}

	return db.Delete(&UserRelation{}, id).Error
}
