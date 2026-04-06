package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"errors"
	"gorm.io/gorm"
)

type Settings struct {
	gorm.Model            `c_migrator:"enabled"`
	RuleInstallerId       int          `json:"rule_installer_id" gorm:"type:int"`
	RuleInstallerRelation UserRelation `json:"read_user_relation" gorm:"references:RuleInstallerId"`
	RuleUserId            int          `json:"rule_user_id" gorm:"type:int"`
	RuleUserRelation      UserRelation `json:"rule_user_relation" gorm:"references:RuleUserId"`
	Rule                  SettingRule  `json:"rule" gorm:"type:jsonb;default:'[]';not null"`
	CreatedAt             time.TimestampWithTimeZoneMicro
	UpdatedAt             time.TimestampWithTimeZoneMicro
	DeletedAt             gorm.DeletedAt `gorm:"index"`
}

var SettingsNotFoundErr = errors.New("settings not found")

// SettingRule TODO: описать
type SettingRule struct{}

func CreateSettings(db *gorm.DB, s *Settings) error {
	return db.Create(s).Error
}

func UpdateSettingsById(db *gorm.DB, s *Settings, id int) error {
	var settings Settings
	result := db.First(&settings, id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return SettingsNotFoundErr
	}

	return db.Save(s).Error
}

func GetSettings(db *gorm.DB) ([]Settings, error) {
	var settings []Settings
	result := db.Find(&settings)

	return settings, result.Error
}

func GetSettingsById(db *gorm.DB, id int) (Settings, error) {
	var settings Settings
	result := db.First(&settings, id)

	return settings, result.Error
}

func DeleteSettingsById(db *gorm.DB, id int) error {
	var settings Settings
	result := db.First(&settings, id)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return SettingsNotFoundErr
	}

	return db.Delete(&Settings{}, id).Error
}
