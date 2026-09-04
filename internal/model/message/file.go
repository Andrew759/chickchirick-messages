package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type File struct {
	gorm.Model `c_migrator:"enabled"`
	Id         int       `json:"id" gorm:"type:int;unique;primaryKey;autoIncrement"`
	MessageId  int       `json:"message_id" gorm:"type:int"`
	Message    Message   `json:"message" gorm:"foreignKey:MessageId"`
	FileUuid   uuid.UUID `json:"file_uuid" gorm:"type:uuid"`
	CreatedAt  time.TimestampWithTimeZoneMicro
	UpdatedAt  time.TimestampWithTimeZoneMicro
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

var FileNotFoundErr = errors.New("file not found")

func CreateFile(ctx context.Context, db *gorm.DB, f *File) error {
	return db.WithContext(ctx).Create(f).Error
}

func UpdateFileById(ctx context.Context, db *gorm.DB, f *File, id int) error {
	var file File
	tx := db.WithContext(ctx)

	result := tx.First(&file, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return FileNotFoundErr
	}

	return tx.Save(f).Error
}

func GetFiles(ctx context.Context, db *gorm.DB) ([]File, error) {
	var files []File
	result := db.WithContext(ctx).Find(&files)

	return files, result.Error
}

func GetFileById(ctx context.Context, db *gorm.DB, id int) (File, error) {
	var file File
	result := db.WithContext(ctx).First(&file, id)

	return file, result.Error
}

func DeleteFileById(ctx context.Context, db *gorm.DB, id int) error {
	var file File
	tx := db.WithContext(ctx)

	result := tx.First(&file, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return FileNotFoundErr
	}

	return tx.Delete(&File{}, id).Error
}
