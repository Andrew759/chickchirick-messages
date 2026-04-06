package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"errors"
	"gorm.io/gorm"
)

type Deleted struct {
	gorm.Model   `c_migrator:"enabled"`
	MessageId    int          `json:"message_id" gorm:"type:int"`
	Message      Message      `json:"message" gorm:"references:MessageId"`
	UserId       int          `json:"user_id" gorm:"type:int"`
	UserRelation UserRelation `json:"user_relation" gorm:"references:UserId"`
	CreatedAt    time.TimestampWithTimeZoneMicro
	UpdatedAt    time.TimestampWithTimeZoneMicro
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

var DeletedNotFoundErr = errors.New("deleted not found")

func CreateDeleted(db *gorm.DB, d *Deleted) error {
	return db.Create(d).Error
}

func UpdateDeletedById(db *gorm.DB, d *Deleted, id int) error {
	var deleted Deleted
	result := db.First(&deleted, id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return DeletedNotFoundErr
	}

	return db.Save(d).Error
}

func GetDeleted(db *gorm.DB) ([]Deleted, error) {
	var deleted []Deleted
	result := db.Find(&deleted)

	return deleted, result.Error
}

func GetDeletedById(db *gorm.DB, id int) (Deleted, error) {
	var deleted Deleted
	result := db.First(&deleted, id)

	return deleted, result.Error
}

func DeleteDeletedById(db *gorm.DB, id int) error {
	var deleted Deleted
	result := db.First(&deleted, id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return DeletedNotFoundErr
	}

	return db.Delete(&Deleted{}, id).Error
}
