package message

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// E2EEIdentity — публичный identity-ключ пользователя (messages-local user_id).
// Сервер хранит только публичный материал; приватные ключи никогда не покидают клиент.
type E2EEIdentity struct {
	UserID           int    `json:"userId" gorm:"primaryKey;column:user_id"`
	IdentityKeySPKI  string `json:"identityKey" gorm:"column:identity_key;type:text;not null"`
	SignedPreKeySPKI string `json:"signedPreKey" gorm:"column:signed_pre_key;type:text;not null"`
	SignedPreKeyID   int    `json:"signedPreKeyId" gorm:"column:signed_pre_key_id;not null"`
	// Подпись SPK identity-ключом (ECDSA P-256, base64). Клиент проверяет.
	SignedPreKeySig string    `json:"signedPreKeySig" gorm:"column:signed_pre_key_sig;type:text;not null"`
	UpdatedAt       time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

func (E2EEIdentity) TableName() string { return "e2ee_identities" }

// E2EEOneTimePreKey — одноразовый prekey (удаляется после выдачи).
type E2EEOneTimePreKey struct {
	ID         int       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID     int       `json:"userId" gorm:"column:user_id;index;not null"`
	KeyID      int       `json:"keyId" gorm:"column:key_id;not null"`
	PublicSPKI string    `json:"publicKey" gorm:"column:public_key;type:text;not null"`
	CreatedAt  time.Time `json:"createdAt" gorm:"column:created_at"`
}

func (E2EEOneTimePreKey) TableName() string { return "e2ee_one_time_prekeys" }

var (
	E2EEIdentityNotFound = errors.New("e2ee identity not found")
	E2EEPreKeyNotFound   = errors.New("e2ee prekey not found")
)

func UpsertE2EEIdentity(ctx context.Context, db *gorm.DB, id *E2EEIdentity) error {
	id.UpdatedAt = time.Now().UTC()
	return db.WithContext(ctx).Save(id).Error
}

func GetE2EEIdentity(ctx context.Context, db *gorm.DB, userID int) (*E2EEIdentity, error) {
	var row E2EEIdentity
	err := db.WithContext(ctx).Where("user_id = ?", userID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, E2EEIdentityNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func ReplaceOneTimePreKeys(ctx context.Context, db *gorm.DB, userID int, keys []E2EEOneTimePreKey) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&E2EEOneTimePreKey{}).Error; err != nil {
			return err
		}
		if len(keys) == 0 {
			return nil
		}
		return tx.Create(&keys).Error
	})
}

func CountOneTimePreKeys(ctx context.Context, db *gorm.DB, userID int) (int64, error) {
	var n int64
	err := db.WithContext(ctx).Model(&E2EEOneTimePreKey{}).Where("user_id = ?", userID).Count(&n).Error
	return n, err
}

// ConsumeOneTimePreKey атомарно выдаёт и удаляет один OPK (если есть).
func ConsumeOneTimePreKey(ctx context.Context, db *gorm.DB, userID int) (*E2EEOneTimePreKey, error) {
	var key E2EEOneTimePreKey
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("user_id = ?", userID).Order("id asc").Limit(1).Find(&key)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 || key.ID == 0 {
			return E2EEPreKeyNotFound
		}
		return tx.Delete(&E2EEOneTimePreKey{}, key.ID).Error
	})
	if err != nil {
		return nil, err
	}
	return &key, nil
}
