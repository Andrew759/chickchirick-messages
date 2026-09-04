package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"context"
	"errors"

	"gorm.io/gorm"
)

type Settings struct {
	gorm.Model            `c_migrator:"enabled"`
	RuleInstallerId       int          `json:"rule_installer_id" gorm:"type:int"`
	RuleInstallerRelation UserRelation `json:"read_user_relation" gorm:"foreignKey:RuleInstallerId;references:UserId"`
	RuleUserId            int          `json:"rule_user_id" gorm:"type:int"`
	RuleUserRelation      UserRelation `json:"rule_user_relation" gorm:"foreignKey:RuleUserId;references:UserId"`
	Rule                  SettingRule  `json:"rule" gorm:"type:jsonb;default:'[]';not null"`
	CreatedAt             time.TimestampWithTimeZoneMicro
	UpdatedAt             time.TimestampWithTimeZoneMicro
	DeletedAt             gorm.DeletedAt `gorm:"index"`
}

var SettingsNotFoundErr = errors.New("settings not found")

// SettingRule TODO: описать
type SettingRule struct{}

func CreateSettings(ctx context.Context, db *gorm.DB, s *Settings) error {
	return db.WithContext(ctx).Create(s).Error
}

func UpdateSettingsById(ctx context.Context, db *gorm.DB, s *Settings, id int) error {
	var settings Settings
	tx := db.WithContext(ctx)

	result := tx.First(&settings, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return SettingsNotFoundErr
	}

	return tx.Save(s).Error
}

func GetSettings(ctx context.Context, db *gorm.DB) ([]Settings, error) {
	var settings []Settings
	result := db.WithContext(ctx).Find(&settings)

	return settings, result.Error
}

func GetSettingsById(ctx context.Context, db *gorm.DB, id int) (Settings, error) {
	var settings Settings
	result := db.WithContext(ctx).First(&settings, id)

	return settings, result.Error
}

func DeleteSettingsById(ctx context.Context, db *gorm.DB, id int) error {
	var settings Settings
	tx := db.WithContext(ctx)

	result := tx.First(&settings, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return SettingsNotFoundErr
	}

	return tx.Delete(&Settings{}, id).Error
}
