package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"context"
	"errors"

	"gorm.io/gorm"
)

type Personal struct {
	gorm.Model        `c_migrator:"enabled"`
	Id                int          `json:"id" gorm:"type:int;unique;primaryKey;autoIncrement"`
	MessageId         int          `json:"message_id" gorm:"type:int"`
	Message           Message      `json:"message" gorm:"foreignKey:MessageId"`
	SenderId          int          `json:"sender_id" gorm:"type:int"`
	SenderRelation    UserRelation `json:"sender_relation" gorm:"foreignKey:SenderId;references:UserId"`
	RecipientId       int          `json:"recipient_id" gorm:"type:int"`
	RecipientRelation UserRelation `json:"recipient_relation" gorm:"foreignKey:RecipientId;references:UserId"`
	CreatedAt         time.TimestampWithTimeZoneMicro
	UpdatedAt         time.TimestampWithTimeZoneMicro
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

var PersonalNotFoundErr = errors.New("personal not found")

func CreatePersonal(ctx context.Context, db *gorm.DB, p *Personal) error {
	return db.WithContext(ctx).Create(p).Error
}

func UpdatePersonalById(ctx context.Context, db *gorm.DB, p *Personal, id int) error {
	var personal Personal
	tx := db.WithContext(ctx)

	result := tx.First(&personal, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return PersonalNotFoundErr
	}

	return tx.Save(p).Error
}

func GetPersonal(ctx context.Context, db *gorm.DB) ([]Personal, error) {
	var personal []Personal
	result := db.WithContext(ctx).Find(&personal)

	return personal, result.Error
}

func GetPersonalById(ctx context.Context, db *gorm.DB, id int) (Personal, error) {
	var personal Personal
	result := db.WithContext(ctx).First(&personal, id)

	return personal, result.Error
}

func DeletePersonalById(ctx context.Context, db *gorm.DB, id int) error {
	var personal Personal
	tx := db.WithContext(ctx)

	result := tx.First(&personal, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return PersonalNotFoundErr
	}

	return tx.Delete(&Personal{}, id).Error
}
