package service

import (
	"chickchirick-messages/internal/gen/messenger"
	"chickchirick-messages/internal/model/message"
	"context"

	"gorm.io/gorm"
)

func CreateMessage(ctx context.Context, db *gorm.DB, req *messenger.SendMessageRequest, userRelation message.UserRelation) (message.Message, error) {
	var msg message.Message
	var err error
	errTx := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		msg = message.Message{
			Text: req.Text,
		}
		if err := tx.Create(&msg).Error; err != nil {
			return err
		}

		//TODO: доработать статусы
		msgStatus := message.Status{
			Model:     gorm.Model{},
			MessageId: msg.Id,
			//ReadUserId:       0,
		}
		if err := tx.Create(&msgStatus).Error; err != nil {
			return err
		}

		var respondMessageId *int = nil
		if req.RespondMessageId != nil {
			val := int(*req.RespondMessageId)
			respondMessageId = &val
		}
		msgMeta := message.Meta{
			MessageId:        msg.Id,
			MessageStatusId:  msgStatus.Id,
			RespondMessageId: respondMessageId,
		}
		if err := tx.Create(&msgMeta).Error; err != nil {
			return err
		}

		msgPersonal := message.Personal{
			MessageId:   msg.Id,
			SenderId:    userRelation.UserId,
			RecipientId: int(req.RecipientId),
		}

		if err := tx.Create(&msgPersonal).Error; err != nil {
			return err
		}
		return nil
	})
	if errTx != nil {
		return msg, errTx
	}
	return msg, err
}
