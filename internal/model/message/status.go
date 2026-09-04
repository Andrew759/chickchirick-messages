package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"context"
	"errors"

	"gorm.io/gorm"
)

type Status struct {
	gorm.Model       `c_migrator:"enabled"`
	Id               int           `json:"id" gorm:"type:int;unique;primaryKey;autoIncrement"`
	MessageId        int           `json:"message_id" gorm:"type:int"`
	Message          Message       `json:"message" gorm:"foreignKey:MessageId"`
	ReadUserId       *int          `json:"read_user_id" gorm:"type:int"`
	ReadUserRelation *UserRelation `json:"read_user_relation" gorm:"foreignKey:ReadUserId;references:UserId"`
	CreatedAt        time.TimestampWithTimeZoneMicro
	UpdatedAt        time.TimestampWithTimeZoneMicro
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

var StatusNotFoundErr = errors.New("status not found")

func CreateStatus(ctx context.Context, db *gorm.DB, s *Status) error {
	return db.WithContext(ctx).Create(s).Error
}

func UpdateStatusById(ctx context.Context, db *gorm.DB, s *Status, id int) error {
	var status Status
	tx := db.WithContext(ctx)

	result := tx.First(&status, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return StatusNotFoundErr
	}

	return tx.Save(s).Error
}

func GetStatus(ctx context.Context, db *gorm.DB) ([]Status, error) {
	var status []Status
	result := db.WithContext(ctx).Find(&status)

	return status, result.Error
}

func GetStatusById(ctx context.Context, db *gorm.DB, id int) (Status, error) {
	var status Status
	result := db.WithContext(ctx).First(&status, id)

	return status, result.Error
}

func DeleteStatusById(ctx context.Context, db *gorm.DB, id int) error {
	var status Status
	tx := db.WithContext(ctx)

	result := tx.First(&status, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return StatusNotFoundErr
	}

	return tx.Delete(&Status{}, id).Error
}
