package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Meta struct {
	gorm.Model       `c_migrator:"enabled" c_migrator_t_name:"message_meta"`
	MessageUuid      uuid.UUID `json:"message_uuid" gorm:"type:uuid;default:gen_random_uuid()"`
	MessageId        int       `json:"message_id" gorm:"type:int"`
	Message          Message   `json:"message" gorm:"foreignKey:MessageId"`
	MessageStatusId  int       `json:"message_status_id" gorm:"type:int"`
	Status           Status    `json:"status" gorm:"foreignKey:MessageStatusId"`
	RespondMessageId *int      `json:"respond_message_id" gorm:"type:int"`
	RespondMessage   *Message  `json:"RespondMessage" gorm:"foreignKey:RespondMessageId"`
	CreatedAt        time.TimestampWithTimeZoneMicro
	UpdatedAt        time.TimestampWithTimeZoneMicro
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

var MetaNotFoundErr = errors.New("meta not found")

func (Meta) TableName() string {
	return "message_meta"
}

func CreateMeta(ctx context.Context, db *gorm.DB, m *Meta) error {
	return db.WithContext(ctx).Create(m).Error
}

func UpdateMetaById(ctx context.Context, db *gorm.DB, m *Meta, id int) error {
	var meta Meta
	tx := db.WithContext(ctx)

	result := tx.First(&meta, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return MetaNotFoundErr
	}

	return tx.Save(m).Error
}

func GetMetas(ctx context.Context, db *gorm.DB) ([]Meta, error) {
	var meta []Meta
	result := db.WithContext(ctx).Find(&meta)

	return meta, result.Error
}

func GetMetaById(ctx context.Context, db *gorm.DB, id int) (Meta, error) {
	var meta Meta
	result := db.WithContext(ctx).First(&meta, id)

	return meta, result.Error
}

func DeleteMetaById(ctx context.Context, db *gorm.DB, id int) error {
	var meta Meta
	tx := db.WithContext(ctx)

	result := tx.First(&meta, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return MetaNotFoundErr
	}

	return tx.Delete(&Meta{}, id).Error
}
