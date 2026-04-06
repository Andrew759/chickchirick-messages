package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"errors"
	"gorm.io/gorm"
)

type Personal struct {
	gorm.Model        `c_migrator:"enabled"`
	Id                int          `json:"id" gorm:"type:int;unique;primaryKey;autoIncrement"`
	MessageId         int          `json:"message_id" gorm:"type:int"`
	Message           Message      `json:"message" gorm:"references:MessageId"`
	SenderId          int          `json:"sender_id" gorm:"type:int"`
	SenderRelation    UserRelation `json:"sender_relation" gorm:"references:SenderId"`
	RecipientId       int          `json:"recipient_id" gorm:"type:int"`
	RecipientRelation UserRelation `json:"recipient_relation" gorm:"references:RecipientId"`
	CreatedAt         time.TimestampWithTimeZoneMicro
	UpdatedAt         time.TimestampWithTimeZoneMicro
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

var PersonalNotFoundErr = errors.New("personal not found")

func CreatePersonal(db *gorm.DB, p *Personal) error {
	return db.Create(p).Error
}

func UpdatePersonalById(db *gorm.DB, p *Personal, id int) error {
	var personal Personal
	result := db.First(&personal, id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return PersonalNotFoundErr
	}

	return db.Save(p).Error
}

func GetPersonal(db *gorm.DB) ([]Personal, error) {
	var personal []Personal
	result := db.Find(&personal)

	return personal, result.Error
}

func GetPersonalById(db *gorm.DB, id int) (Personal, error) {
	var personal Personal
	result := db.First(&personal, id)

	return personal, result.Error
}

func DeletePersonalById(db *gorm.DB, id int) error {
	var personal Personal
	result := db.First(&personal, id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return PersonalNotFoundErr
	}

	return db.Delete(&Personal{}, id).Error
}
