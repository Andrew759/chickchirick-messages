package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TODO: добавить констрейнт на уникальынй user_uuid
type UserRelation struct {
	UserId    int       `json:"id" gorm:"type:int;unique;primaryKey;autoIncrement" c_migrator:"enabled"`
	UserUuid  uuid.UUID `json:"user_uuid" gorm:"type:uuid;unique"`
	CreatedAt time.TimestampWithTimeZoneMicro
	UpdatedAt time.TimestampWithTimeZoneMicro
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

var UserRelationNotFoundErr = errors.New("user relation not found")

func CreateUserRelation(ctx context.Context, db *gorm.DB, ur *UserRelation) error {
	return db.WithContext(ctx).Create(ur).Error
}

func UpdateUserRelationById(ctx context.Context, db *gorm.DB, ur *UserRelation, id int) error {
	var userRelation UserRelation
	tx := db.WithContext(ctx)

	result := tx.First(&userRelation, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return UserRelationNotFoundErr
	}

	return tx.Save(ur).Error
}

func GetUserRelation(ctx context.Context, db *gorm.DB) ([]UserRelation, error) {
	var userRelation []UserRelation
	result := db.WithContext(ctx).Find(&userRelation)

	return userRelation, result.Error
}

func GetUserRelationById(ctx context.Context, db *gorm.DB, id int) (UserRelation, error) {
	var userRelation UserRelation
	result := db.WithContext(ctx).First(&userRelation, id)

	return userRelation, result.Error
}

func GetUserRelationByUuid(ctx context.Context, db *gorm.DB, uuid string) (UserRelation, error) {
	var userRelation UserRelation
	result := db.WithContext(ctx).Where("user_uuid = ?", uuid).First(&userRelation)

	//TODO: доработать ошибки
	return userRelation, result.Error
}

func DeleteUserRelationById(ctx context.Context, db *gorm.DB, id int) error {
	var userRelation UserRelation
	tx := db.WithContext(ctx)

	result := tx.First(&userRelation, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return UserRelationNotFoundErr
	}

	return tx.Delete(&UserRelation{}, id).Error
}
