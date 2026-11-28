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
	Type             grpc.MessageType            `json:"type"`
	Message          *messenger.Message          `json:"message,omitempty"`
	GroupId          string                      `json:"group_id,omitempty"`
	Group            *GroupAddedPayload          `json:"group,omitempty"`
	MembersChanged   *GroupMembersChangedPayload `json:"members_changed,omitempty"`
	Typing           *GroupTypingPayload         `json:"typing,omitempty"`
	DeletedMessageID int64                       `json:"deleted_message_id,omitempty"`
	Edited           bool                        `json:"edited,omitempty"`
}

// GroupTypingPayload — кто печатает в группе.
type GroupTypingPayload struct {
	GroupUuid string `json:"group_uuid"`
	SenderID  int64  `json:"sender_id"`
	Typing    bool   `json:"typing"`
}

// GroupMembersChangedPayload — состав группы изменился (E2EE: redistribute / rotate).
type GroupMembersChangedPayload struct {
	GroupUuid      string `json:"group_uuid"`
	Title          string `json:"title,omitempty"`
	AddedUserIds   []int  `json:"added_user_ids,omitempty"`
	RemovedUserIds []int  `json:"removed_user_ids,omitempty"`
	MemberCount    int    `json:"member_count"`
	ActorUserId    int    `json:"actor_user_id"`
}

// GroupAddedPayload — payload для realtime «вас добавили в группу».
type GroupAddedPayload struct {
	GroupUuid      string  `json:"group_uuid"`
	Title          string  `json:"title"`
	AvatarFileUuid *string `json:"avatar_file_uuid,omitempty"`
	CreatedBy      int     `json:"created_by"`
	MemberCount    int     `json:"member_count"`
}

func ChatGroupToPayload(g message.GroupMeta, memberCount int) GroupAddedPayload {
	p := GroupAddedPayload{
		GroupUuid:   g.GroupUuid.String(),
		Title:       g.Title,
		CreatedBy:   g.CreatedBy,
		MemberCount: memberCount,
	}
	if g.AvatarFileUuid != nil {
		s := g.AvatarFileUuid.String()
		p.AvatarFileUuid = &s
	}
	return p
}

// PublishGroupAdded уведомляет пользователей, что их добавили в группу.
func PublishGroupAdded(ctx context.Context, rdb *redis.Client, userIds []int, payload GroupAddedPayload) error {
	if rdb == nil || len(userIds) == 0 {
		return nil
	}
	eventData, err := json.Marshal(GroupMessageEvent{
		Type:  grpc.TypeGroupAdded,
		Group: &payload,
	})
	if err != nil {
		return err
	}
	seen := map[int]struct{}{}
	for _, uid := range userIds {
		if uid <= 0 {
			continue
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		if err := rdb.Publish(ctx, fmt.Sprintf("user_events_%d", uid), eventData).Err(); err != nil {
			return err
		}
	}
	return nil
}

// PublishGroupMembersChanged шлёт realtime тем, кто уже в группе (владельцы ключа перераздают E2EE).
func PublishGroupMembersChanged(ctx context.Context, rdb *redis.Client, userIds []int, payload GroupMembersChangedPayload) error {
	if rdb == nil || len(userIds) == 0 {
		return nil
	}
	eventData, err := json.Marshal(GroupMessageEvent{
		Type:           grpc.TypeGroupMembersChanged,
		MembersChanged: &payload,
	})
	if err != nil {
		return err
	}
	seen := map[int]struct{}{}
	for _, uid := range userIds {
		if uid <= 0 {
			continue
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		if err := rdb.Publish(ctx, fmt.Sprintf("user_events_%d", uid), eventData).Err(); err != nil {
			return err
		}
	}
	return nil
}

// CreateGroupChat создаёт группу, добавляет владельца и участников.
func CreateGroupChat(ctx context.Context, db *gorm.DB, creator message.UserRelation, in CreateGroupInput) (message.GroupMeta, []message.GroupMember, error) {
	var gMeta message.GroupMeta
	var members []message.GroupMember

	title := in.Title
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		gMeta = message.GroupMeta{
			GroupUuid:      uuid.New(),
			Title:          title,
			AvatarFileUuid: in.AvatarFileUuid,
			CreatedBy:      creator.UserId,
		}
		if err := message.CreateGroupMeta(ctx, tx, &gMeta); err != nil {
			return err
		}

		owner := message.GroupMember{
			GroupId: gMeta.Id,
			UserId:  creator.UserId,
			Role:    message.GroupRoleOwner,
		}
		if err := message.CreateGroupMember(ctx, tx, &owner); err != nil {
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
			m := message.GroupMember{
				GroupId: gMeta.Id,
				UserId:  uid,
				Role:    message.GroupRoleMember,
			}
			if err := message.CreateGroupMember(ctx, tx, &m); err != nil {
				return err
			}
			members = append(members, m)
		}
		return nil
	})

	return gMeta, members, err
}

// CreateGroupMessage пишет сообщение в группу и рассылает всем участникам через Redis.
func CreateGroupMessage(ctx context.Context, db *gorm.DB, rdb *redis.Client, sender message.UserRelation, groupUuid uuid.UUID, text string, respondMessageId *int64) (message.Message, message.GroupMeta, error) {
	var msg message.Message
	var group message.GroupMeta

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		g, err := message.GetGroupMetaByUuid(ctx, tx, groupUuid)
		if err != nil {
			return err
		}
		group = g

		ok, err := message.IsGroupMember(ctx, tx, g.Id, sender.UserId)
		if err != nil {
			return err
		}
		if !ok {
			return message.GroupForbiddenErr
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
		_ = PublishGroupMessage(ctx, db, rdb, group, msg, sender.UserId, respondMessageId)
	}
	return msg, group, nil
}

func PublishGroupMessage(ctx context.Context, db *gorm.DB, rdb *redis.Client, group message.GroupMeta, msg message.Message, senderId int, respondMessageId *int64) error {
	memberIds, err := message.ListGroupMemberUserIds(ctx, db, group.Id)
	if err != nil {
		return err
	}

	// group_id несём в Redis-событии (не через recipient_id=0)
	protoMsg := &messenger.Message{
		Id:               int64(msg.Id),
		SenderId:         int64(senderId),
		Text:             msg.Text,
		CreatedAt:        timestamppb.Now(),
		RespondMessageId: respondMessageId,
	}

	eventData, err := json.Marshal(GroupMessageEvent{
		Type:    grpc.TypeNewMessage,
		Message: protoMsg,
		GroupId: group.GroupUuid.String(),
	})
	if err != nil {
		return err
	}

	seen := map[int]struct{}{}
	for _, uid := range memberIds {
		if uid <= 0 {
			continue
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		if err := rdb.Publish(ctx, fmt.Sprintf("user_events_%d", uid), eventData).Err(); err != nil {
			return err
		}
	}
	return nil
}

// PublishGroupTyping — typing fan-out всем участникам группы (кроме отправителя).
func PublishGroupTyping(ctx context.Context, db *gorm.DB, rdb *redis.Client, groupUuid uuid.UUID, senderId int, typing bool) error {
	if rdb == nil {
		return nil
	}
	g, err := message.GetGroupMetaByUuid(ctx, db, groupUuid)
	if err != nil {
		return err
	}
	ok, err := message.IsGroupMember(ctx, db, g.Id, senderId)
	if err != nil {
		return err
	}
	if !ok {
		return message.GroupForbiddenErr
	}
	memberIds, err := message.ListGroupMemberUserIds(ctx, db, g.Id)
	if err != nil {
		return err
	}
	eventData, err := json.Marshal(GroupMessageEvent{
		Type:    grpc.TypeTyping,
		GroupId: g.GroupUuid.String(),
		Typing: &GroupTypingPayload{
			GroupUuid: g.GroupUuid.String(),
			SenderID:  int64(senderId),
			Typing:    typing,
		},
	})
	if err != nil {
		return err
	}
	seen := map[int]struct{}{}
	for _, uid := range memberIds {
		if uid <= 0 || uid == senderId {
			continue
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		if err := rdb.Publish(ctx, fmt.Sprintf("user_events_%d", uid), eventData).Err(); err != nil {
			return err
		}
	}
	return nil
}

// DeleteGroupMessage удаляет сообщение в группе. Только отправитель.
func UpdateGroupMessage(ctx context.Context, db *gorm.DB, rdb *redis.Client, actor message.UserRelation, groupUuid uuid.UUID, messageID int, text string) error {
	g, err := message.GetGroupMetaByUuid(ctx, db, groupUuid)
	if err != nil {
		return err
	}
	ok, err := message.IsGroupMember(ctx, db, g.Id, actor.UserId)
	if err != nil {
		return err
	}
	if !ok {
		return message.GroupForbiddenErr
	}
	link, err := message.GetGroupByMessageId(ctx, db, messageID)
	if err != nil {
		return err
	}
	if link.GroupId != g.GroupUuid {
		return message.GroupNotFoundErr
	}
	if link.SenderId != actor.UserId {
		return fmt.Errorf("forbidden: only sender can edit the message")
	}
	if err := message.UpdateMessageTextById(ctx, db, messageID, text); err != nil {
		return err
	}
	updated, err := message.GetMessageById(ctx, db, messageID)
	if err != nil {
		return err
	}
	if rdb == nil {
		return nil
	}
	memberIds, err := message.ListGroupMemberUserIds(ctx, db, g.Id)
	if err != nil {
		return err
	}
	protoMsg := &messenger.Message{
		Id: int64(messageID), SenderId: int64(actor.UserId), Text: updated.Text,
		CreatedAt: timestamppb.New(updated.CreatedAt.Time),
	}
	eventData, err := json.Marshal(GroupMessageEvent{
		Type: grpc.TypeNewMessage, Message: protoMsg, GroupId: g.GroupUuid.String(), Edited: true,
	})
	if err != nil {
		return err
	}
	seen := map[int]struct{}{}
	for _, uid := range memberIds {
		if uid <= 0 {
			continue
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		if err := rdb.Publish(ctx, fmt.Sprintf("user_events_%d", uid), eventData).Err(); err != nil {
			return err
		}
	}
	return nil
}

func DeleteGroupMessage(
	ctx context.Context,
	db *gorm.DB,
	rdb *redis.Client,
	actor message.UserRelation,
	groupUuid uuid.UUID,
	messageID int,
) error {
	g, err := message.GetGroupMetaByUuid(ctx, db, groupUuid)
	if err != nil {
		return err
	}
	ok, err := message.IsGroupMember(ctx, db, g.Id, actor.UserId)
	if err != nil {
		return err
	}
	if !ok {
		return message.GroupForbiddenErr
	}

	link, err := message.GetGroupByMessageId(ctx, db, messageID)
	if err != nil {
		return err
	}
	if link.GroupId != g.GroupUuid {
		return message.GroupNotFoundErr
	}
	if link.SenderId != actor.UserId {
		return fmt.Errorf("forbidden: only sender can delete the message")
	}

	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := message.DeleteMessageById(ctx, tx, messageID); err != nil {
			return err
		}
		_ = message.DeleteGroupById(ctx, tx, link.Id)
		d := message.Deleted{MessageId: messageID, UserId: actor.UserId}
		_ = message.CreateDeleted(ctx, tx, &d)
		return nil
	})
	if err != nil {
		return err
	}

	if rdb != nil {
		_ = PublishGroupMessageDeleted(ctx, db, rdb, g, int64(messageID))
	}
	return nil
}

func PublishGroupMessageDeleted(ctx context.Context, db *gorm.DB, rdb *redis.Client, group message.GroupMeta, messageID int64) error {
	if rdb == nil {
		return nil
	}
	memberIds, err := message.ListGroupMemberUserIds(ctx, db, group.Id)
	if err != nil {
		return err
	}
	eventData, err := json.Marshal(GroupMessageEvent{
		Type:             grpc.TypeDelete,
		GroupId:          group.GroupUuid.String(),
		DeletedMessageID: messageID,
	})
	if err != nil {
		return err
	}
	seen := map[int]struct{}{}
	for _, uid := range memberIds {
		if uid <= 0 {
			continue
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		if err := rdb.Publish(ctx, fmt.Sprintf("user_events_%d", uid), eventData).Err(); err != nil {
			return err
		}
	}
	return nil
}
