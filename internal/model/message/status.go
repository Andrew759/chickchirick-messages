package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"errors"

	"gorm.io/gorm"
)

type Status struct {
	gorm.Model       `c_migrator:"enabled"`
	Id               int                             `json:"id" gorm:"type:int;unique;primaryKey;autoIncrement"`
	MessageId        int                             `json:"message_id" gorm:"type:int"`
	Message          Message                         `json:"message" gorm:"references:MessageId"`
	ReadUserId       int                             `json:"read_user_id" gorm:"type:int"`
	ReadUserRelation UserRelation                    `json:"read_user_relation" gorm:"references:ReadUserId"`
	Date             time.TimestampWithTimeZoneMicro `json:"date" gorm:"type:timestamp without time zone"`
	CreatedAt        time.TimestampWithTimeZoneMicro
	UpdatedAt        time.TimestampWithTimeZoneMicro
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

var StatusNotFoundErr = errors.New("status not found")

func CreateStatus(db *gorm.DB, s *Status) error {
	return db.Create(s).Error
}

func UpdateStatusById(db *gorm.DB, s *Status, id int) error {
	var status Status
	result := db.First(&status, id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return StatusNotFoundErr
	}

	return db.Save(s).Error
}

func GetStatus(db *gorm.DB) ([]Status, error) {
	var status []Status
	result := db.Find(&status)

	return status, result.Error
}

func GetStatusById(db *gorm.DB, id int) (Status, error) {
	var status Status
	result := db.First(&status, id)

	return status, result.Error
}

func DeleteStatusById(db *gorm.DB, id int) error {
	var status Status
	result := db.First(&status, id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return StatusNotFoundErr
	}

	return db.Delete(&Status{}, id).Error
}
