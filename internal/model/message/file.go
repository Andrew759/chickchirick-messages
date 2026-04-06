package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type File struct {
	gorm.Model `c_migrator:"enabled"`
	Id         int       `json:"id" gorm:"type:int;unique;primaryKey;autoIncrement"`
	MessageId  int       `json:"message_id" gorm:"type:int"`
	Message    Message   `json:"message" gorm:"references:MessageId"`
	FileUuid   uuid.UUID `json:"file_uuid" gorm:"type:uuid"`
	CreatedAt  time.TimestampWithTimeZoneMicro
	UpdatedAt  time.TimestampWithTimeZoneMicro
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

var FileNotFoundErr = errors.New("file not found")

func CreateFile(db *gorm.DB, f *File) error {
	return db.Create(f).Error
}

func UpdateFileById(db *gorm.DB, f *File, id int) error {
	var file File
	result := db.First(&file, id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return FileNotFoundErr
	}

	return db.Save(f).Error
}

func GetFiles(db *gorm.DB) ([]File, error) {
	var files []File
	result := db.Find(&files)

	return files, result.Error
}

func GetFileById(db *gorm.DB, id int) (File, error) {
	var file File
	result := db.First(&file, id)

	return file, result.Error
}

func DeleteFileById(db *gorm.DB, id int) error {
	var file File
	result := db.First(&file, id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return FileNotFoundErr
	}

	return db.Delete(&File{}, id).Error
}
