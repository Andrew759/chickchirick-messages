package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"context"
	"errors"

	"gorm.io/gorm"
)

type Deleted struct {
	gorm.Model   `c_migrator:"enabled"`
	Id           int          `json:"id" gorm:"type:int;unique;primaryKey;autoIncrement"`
	MessageId    int          `json:"message_id" gorm:"type:int"`
	Message      Message      `json:"message" gorm:"foreignKey:MessageId"`
	UserId       int          `json:"user_id" gorm:"type:int"`
	UserRelation UserRelation `json:"user_relation" gorm:"foreignKey:UserId;references:UserId"`
	CreatedAt    time.TimestampWithTimeZoneMicro
	UpdatedAt    time.TimestampWithTimeZoneMicro
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

var DeletedNotFoundErr = errors.New("deleted not found")

func CreateDeleted(ctx context.Context, db *gorm.DB, d *Deleted) error {
	return db.WithContext(ctx).Create(d).Error
}

func UpdateDeletedById(ctx context.Context, db *gorm.DB, d *Deleted, id int) error {
	var deleted Deleted
	tx := db.WithContext(ctx)

	result := tx.First(&deleted, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return DeletedNotFoundErr
	}

	return tx.Save(d).Error
}

func GetDeleted(ctx context.Context, db *gorm.DB) ([]Deleted, error) {
	var deleted []Deleted
	result := db.WithContext(ctx).Find(&deleted)

	return deleted, result.Error
}

func GetDeletedById(ctx context.Context, db *gorm.DB, id int) (Deleted, error) {
	var deleted Deleted
	result := db.WithContext(ctx).First(&deleted, id)

	return deleted, result.Error
}

func DeleteDeletedById(ctx context.Context, db *gorm.DB, id int) error {
	var deleted Deleted
	tx := db.WithContext(ctx)

	result := tx.First(&deleted, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return DeletedNotFoundErr
	}

	return tx.Delete(&Deleted{}, id).Error
}
