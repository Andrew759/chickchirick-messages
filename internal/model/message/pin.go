package message

import (
	"chickchirick-messages/pkg/chirik_gorm_tweaks/time"
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Pin — закреплённое сообщение в области чата (ЛС / группа / канал).
type Pin struct {
	gorm.Model `c_migrator:"enabled" c_migrator_t_name:"message_pins"`
	Id         int     `json:"id" gorm:"type:int;unique;primaryKey;autoIncrement"`
	MessageId  int     `json:"message_id" gorm:"type:int;not null;index"`
	Message    Message `json:"message,omitempty" gorm:"foreignKey:MessageId"`

	// Scope: ровно одно из полей заполнено
	PeerUserLow  *int       `json:"peer_user_low,omitempty" gorm:"type:int;index"`  // min(userA,userB) для ЛС
	PeerUserHigh *int       `json:"peer_user_high,omitempty" gorm:"type:int;index"` // max(userA,userB)
	GroupUuid    *uuid.UUID `json:"group_uuid,omitempty" gorm:"type:uuid;index"`
	ChannelUuid  *uuid.UUID `json:"channel_uuid,omitempty" gorm:"type:uuid;index"`

	PinnedBy  int `json:"pinned_by" gorm:"type:int;not null"`
	CreatedAt time.TimestampWithTimeZoneMicro
	UpdatedAt time.TimestampWithTimeZoneMicro
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Pin) TableName() string { return "message_pins" }

var PinNotFoundErr = errors.New("pin not found")
var PinScopeErr = errors.New("invalid pin scope")

func CreatePin(ctx context.Context, db *gorm.DB, p *Pin) error {
	return db.WithContext(ctx).Create(p).Error
}

func DeletePinByMessageAndScope(ctx context.Context, db *gorm.DB, messageID int, peerLow, peerHigh *int, groupUuid, channelUuid *uuid.UUID) error {
	q := db.WithContext(ctx).Unscoped().Where("message_id = ?", messageID)
	q = applyPinScope(q, peerLow, peerHigh, groupUuid, channelUuid)
	res := q.Delete(&Pin{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return PinNotFoundErr
	}
	return nil
}

func FindPin(ctx context.Context, db *gorm.DB, messageID int, peerLow, peerHigh *int, groupUuid, channelUuid *uuid.UUID) (Pin, error) {
	var p Pin
	q := db.WithContext(ctx).Where("message_id = ? AND deleted_at IS NULL", messageID)
	q = applyPinScope(q, peerLow, peerHigh, groupUuid, channelUuid)
	err := q.First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return p, PinNotFoundErr
	}
	return p, err
}

func ListPinsByScope(ctx context.Context, db *gorm.DB, peerLow, peerHigh *int, groupUuid, channelUuid *uuid.UUID) ([]Pin, error) {
	var rows []Pin
	q := db.WithContext(ctx).Where("deleted_at IS NULL")
	q = applyPinScope(q, peerLow, peerHigh, groupUuid, channelUuid)
	err := q.Order("id DESC").Find(&rows).Error
	return rows, err
}

func applyPinScope(q *gorm.DB, peerLow, peerHigh *int, groupUuid, channelUuid *uuid.UUID) *gorm.DB {
	if groupUuid != nil {
		return q.Where("group_uuid = ?", *groupUuid)
	}
	if channelUuid != nil {
		return q.Where("channel_uuid = ?", *channelUuid)
	}
	if peerLow != nil && peerHigh != nil {
		return q.Where("peer_user_low = ? AND peer_user_high = ?", *peerLow, *peerHigh)
	}
	return q.Where("1 = 0")
}
