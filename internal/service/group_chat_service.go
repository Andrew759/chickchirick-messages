package service

import (
	"chickchirick-messages/internal/enum/grpc"
	"chickchirick-messages/internal/gen/messenger"
	"chickchirick-messages/internal/model/message"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type CreateGroupInput struct {
	Title          string
	AvatarFileUuid *uuid.UUID
	MemberUserIds  []int // messages-local ids, без создателя
}

type GroupMessageEvent struct {
	Type             grpc.MessageType   `json:"type"`
	Message          *messenger.Message `json:"message,omitempty"`
	GroupId          string             `json:"group_id,omitempty"`
	DeletedMessageID int64              `json:"deleted_message_id,omitempty"`
}

// CreateGroupChat создаёт группу, добавляет владельца и участников.
func CreateGroupChat(ctx context.Context, db *gorm.DB, creator message.UserRelation, in CreateGroupInput) (message.ChatGroup, []message.ChatGroupMember, error) {
	var group message.ChatGroup
	var members []message.ChatGroupMember

	title := in.Title
	if title == "" {
		title = "Группа"
	}

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		group = message.ChatGroup{
			GroupUuid:      uuid.New(),
			Title:          title,
			AvatarFileUuid: in.AvatarFileUuid,
			CreatedBy:      creator.UserId,
		}
		if err := message.CreateChatGroup(ctx, tx, &group); err != nil {
			return err
		}

		owner := message.ChatGroupMember{
			GroupId: group.Id,
			UserId:  creator.UserId,
			Role:    message.GroupRoleOwner,
		}
		if err := message.CreateChatGroupMember(ctx, tx, &owner); err != nil {
			return err
		}
		members = append(members, owner)

		seen := map[int]struct{}{creator.UserId: {}}
		for _, uid := range in.MemberUserIds {
			if uid <= 0 {
				continue
			}
			if _, ok := seen[uid]; ok {
				continue
			}
			seen[uid] = struct{}{}
			m := message.ChatGroupMember{
				GroupId: group.Id,
				UserId:  uid,
				Role:    message.GroupRoleMember,
			}
			if err := message.CreateChatGroupMember(ctx, tx, &m); err != nil {
				return err
			}
			members = append(members, m)
		}
		return nil
	})

	return group, members, err
}

// CreateGroupMessage пишет сообщение в группу и рассылает всем участникам через Redis.
func CreateGroupMessage(ctx context.Context, db *gorm.DB, rdb *redis.Client, sender message.UserRelation, groupUuid uuid.UUID, text string, respondMessageId *int64) (message.Message, message.ChatGroup, error) {
	var msg message.Message
	var group message.ChatGroup

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		g, err := message.GetChatGroupByUuid(ctx, tx, groupUuid)
		if err != nil {
			return err
		}
		group = g

		ok, err := message.IsGroupMember(ctx, tx, g.Id, sender.UserId)
		if err != nil {
			return err
		}
		if !ok {
			return message.ChatGroupForbiddenErr
		}

		msg = message.Message{Text: text}
		if err := tx.Create(&msg).Error; err != nil {
			return err
		}

		status := message.Status{MessageId: msg.Id}
		if err := tx.Create(&status).Error; err != nil {
			return err
		}

		var respond *int
		if respondMessageId != nil {
			v := int(*respondMessageId)
			respond = &v
		}
		meta := message.Meta{
			MessageId:        msg.Id,
			MessageStatusId:  status.Id,
			RespondMessageId: respond,
		}
		if err := tx.Create(&meta).Error; err != nil {
			return err
		}

		link := message.Group{
			MessageId: msg.Id,
			GroupId:   g.GroupUuid,
			SenderId:  sender.UserId,
		}
		if err := message.CreateGroup(ctx, tx, &link); err != nil {
			return err
		}

		// touch updated_at группы
		_ = tx.Model(&g).UpdateColumn("updated_at", gorm.Expr("NOW()")).Error
		return nil
	})
	if err != nil {
		return msg, group, err
	}

	if rdb != nil {
		_ = PublishGroupMessage(ctx, db, rdb, group, msg, sender.UserId)
	}
	return msg, group, nil
}

func PublishGroupMessage(ctx context.Context, db *gorm.DB, rdb *redis.Client, group message.ChatGroup, msg message.Message, senderId int) error {
	memberIds, err := message.ListGroupMemberUserIds(ctx, db, group.Id)
	if err != nil {
		return err
	}

	protoMsg := &messenger.Message{
		Id:          int64(msg.Id),
		SenderId:    int64(senderId),
		RecipientId: 0,
		Text:        msg.Text,
		CreatedAt:   timestamppb.Now(),
	}

	eventData, err := json.Marshal(GroupMessageEvent{
		Type:    grpc.TypeNewMessage,
		Message: protoMsg,
		GroupId: group.GroupUuid.String(),
	})
	if err != nil {
		return err
	}

	for _, uid := range memberIds {
		if err := rdb.Publish(ctx, fmt.Sprintf("user_events_%d", uid), eventData).Err(); err != nil {
			return err
		}
	}
	return nil
}
