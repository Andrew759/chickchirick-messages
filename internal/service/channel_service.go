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

type CreateChannelInput struct {
	Title          string
	Description    string
	AvatarFileUuid *uuid.UUID
	IsPublic       bool
	E2EEEnabled    bool
	// MemberUserIds — initial subscribers/admins (без создателя); создатель всегда owner
	MemberUserIds []int
}

type ChannelMessageEvent struct {
	Type             grpc.MessageType              `json:"type"`
	Message          *messenger.Message            `json:"message,omitempty"`
	ChannelId        string                        `json:"channel_id,omitempty"`
	Channel          *ChannelAddedPayload          `json:"channel,omitempty"`
	MembersChanged   *ChannelMembersChangedPayload `json:"channel_members_changed,omitempty"`
	Typing           *ChannelTypingPayload         `json:"typing,omitempty"`
	DeletedMessageID int64                         `json:"deleted_message_id,omitempty"`
	Edited           bool                          `json:"edited,omitempty"`
}

type ChannelTypingPayload struct {
	ChannelUuid string `json:"channel_uuid"`
	SenderID    int64  `json:"sender_id"`
	Typing      bool   `json:"typing"`
}

// ChannelMembersChangedPayload — состав канала изменился (E2EE: redistribute / rotate ChannelKey).
type ChannelMembersChangedPayload struct {
	ChannelUuid    string `json:"channel_uuid"`
	Title          string `json:"title,omitempty"`
	AddedUserIds   []int  `json:"added_user_ids,omitempty"`
	RemovedUserIds []int  `json:"removed_user_ids,omitempty"`
	MemberCount    int    `json:"member_count"`
	ActorUserId    int    `json:"actor_user_id"`
	E2EEEnabled    bool   `json:"e2ee_enabled"`
}

type ChannelAddedPayload struct {
	ChannelUuid    string  `json:"channel_uuid"`
	Title          string  `json:"title"`
	Description    string  `json:"description,omitempty"`
	AvatarFileUuid *string `json:"avatar_file_uuid,omitempty"`
	IsPublic       bool    `json:"is_public"`
	E2EEEnabled    bool    `json:"e2ee_enabled"`
	CreatedBy      int     `json:"created_by"`
	MemberCount    int     `json:"member_count"`
}

func ChannelToPayload(ch message.Channel, memberCount int) ChannelAddedPayload {
	p := ChannelAddedPayload{
		ChannelUuid: ch.ChannelUuid.String(),
		Title:       ch.Title,
		Description: ch.Description,
		IsPublic:    ch.IsPublic,
		E2EEEnabled: ch.E2EEEnabled,
		CreatedBy:   ch.CreatedBy,
		MemberCount: memberCount,
	}
	if ch.AvatarFileUuid != nil {
		s := ch.AvatarFileUuid.String()
		p.AvatarFileUuid = &s
	}
	return p
}

func PublishChannelAdded(ctx context.Context, rdb *redis.Client, userIds []int, payload ChannelAddedPayload) error {
	if rdb == nil || len(userIds) == 0 {
		return nil
	}
	eventData, err := json.Marshal(ChannelMessageEvent{
		Type:    grpc.TypeChannelAdded,
		Channel: &payload,
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

// PublishChannelMembersChanged — realtime тем, у кого уже есть ChannelKey (E2EE rotate).
func PublishChannelMembersChanged(ctx context.Context, rdb *redis.Client, userIds []int, payload ChannelMembersChangedPayload) error {
	if rdb == nil || len(userIds) == 0 {
		return nil
	}
	eventData, err := json.Marshal(ChannelMessageEvent{
		Type:           grpc.TypeChannelMembersChanged,
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

// CreateChannel создаёт канал, владельца и начальных участников.
func CreateChannel(ctx context.Context, db *gorm.DB, creator message.UserRelation, in CreateChannelInput) (message.Channel, []message.ChannelMember, error) {
	var ch message.Channel
	var members []message.ChannelMember

	title := in.Title
	if title == "" {
		title = "Канал"
	}

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ch = message.Channel{
			ChannelUuid:    uuid.New(),
			Title:          title,
			Description:    in.Description,
			AvatarFileUuid: in.AvatarFileUuid,
			IsPublic:       in.IsPublic,
			E2EEEnabled:    in.E2EEEnabled,
			CreatedBy:      creator.UserId,
		}
		// По умолчанию E2EE включён (безопасный default для приватных).
		// Для публичных без шифрования клиент передаёт e2ee_enabled=false.
		if !in.IsPublic && !in.E2EEEnabled {
			// оставляем как передал клиент; default true задаётся на уровне JSON/API
		}
		if err := message.CreateChannel(ctx, tx, &ch); err != nil {
			return err
		}

		owner := message.ChannelMember{
			ChannelId: ch.Id,
			UserId:    creator.UserId,
			Role:      message.ChannelRoleOwner,
		}
		if err := message.CreateChannelMember(ctx, tx, &owner); err != nil {
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
			m := message.ChannelMember{
				ChannelId: ch.Id,
				UserId:    uid,
				Role:      message.ChannelRoleSubscriber,
			}
			if err := message.CreateChannelMember(ctx, tx, &m); err != nil {
				return err
			}
			members = append(members, m)
		}
		return nil
	})

	return ch, members, err
}

// CreateChannelMessage — пост в канал. Только owner/admin. Text = ciphertext при E2EE.
func CreateChannelMessage(
	ctx context.Context,
	db *gorm.DB,
	rdb *redis.Client,
	sender message.UserRelation,
	channelUuid uuid.UUID,
	text string,
	respondMessageId *int64,
) (message.Message, message.Channel, error) {
	var msg message.Message
	var ch message.Channel

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		c, err := message.GetChannelByUuid(ctx, tx, channelUuid)
		if err != nil {
			return err
		}
		ch = c

		canPost, err := message.CanPostToChannel(ctx, tx, c.Id, sender.UserId)
		if err != nil {
			return err
		}
		if !canPost {
			return message.ChannelPostForbiddenErr
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

		link := message.ChannelMsg{
			MessageId: msg.Id,
			ChannelId: c.ChannelUuid,
			SenderId:  sender.UserId,
		}
		if err := message.CreateChannelMsg(ctx, tx, &link); err != nil {
			return err
		}

		_ = tx.Model(&c).UpdateColumn("updated_at", gorm.Expr("NOW()")).Error
		return nil
	})
	if err != nil {
		return msg, ch, err
	}

	if rdb != nil {
		_ = PublishChannelMessage(ctx, db, rdb, ch, msg, sender.UserId, respondMessageId)
	}
	return msg, ch, nil
}

func PublishChannelMessage(ctx context.Context, db *gorm.DB, rdb *redis.Client, ch message.Channel, msg message.Message, senderId int, respondMessageId *int64) error {
	memberIds, err := message.ListChannelMemberUserIds(ctx, db, ch.Id)
	if err != nil {
		return err
	}

	protoMsg := &messenger.Message{
		Id:               int64(msg.Id),
		SenderId:         int64(senderId),
		Text:             msg.Text,
		CreatedAt:        timestamppb.Now(),
		RespondMessageId: respondMessageId,
	}

	eventData, err := json.Marshal(ChannelMessageEvent{
		Type:      grpc.TypeNewMessage,
		Message:   protoMsg,
		ChannelId: ch.ChannelUuid.String(),
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

// PublishChannelTyping — typing fan-out подписчикам (кроме отправителя).
func PublishChannelTyping(ctx context.Context, db *gorm.DB, rdb *redis.Client, channelUuid uuid.UUID, senderId int, typing bool) error {
	if rdb == nil {
		return nil
	}
	ch, err := message.GetChannelByUuid(ctx, db, channelUuid)
	if err != nil {
		return err
	}
	ok, err := message.IsChannelMember(ctx, db, ch.Id, senderId)
	if err != nil {
		return err
	}
	if !ok {
		return message.ChannelForbiddenErr
	}
	// Писать typing могут только те, кто может постить
	canPost, err := message.CanPostToChannel(ctx, db, ch.Id, senderId)
	if err != nil {
		return err
	}
	if !canPost {
		return message.ChannelPostForbiddenErr
	}

	memberIds, err := message.ListChannelMemberUserIds(ctx, db, ch.Id)
	if err != nil {
		return err
	}
	eventData, err := json.Marshal(ChannelMessageEvent{
		Type:      grpc.TypeTyping,
		ChannelId: ch.ChannelUuid.String(),
		Typing: &ChannelTypingPayload{
			ChannelUuid: ch.ChannelUuid.String(),
			SenderID:    int64(senderId),
			Typing:      typing,
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

// SubscribeToChannel — самоподписка на публичный канал (или invite-flow для приватного на уровне API).
func SubscribeToChannel(ctx context.Context, db *gorm.DB, rdb *redis.Client, user message.UserRelation, channelUuid uuid.UUID) (message.Channel, error) {
	ch, err := message.GetChannelByUuid(ctx, db, channelUuid)
	if err != nil {
		return ch, err
	}
	if !ch.IsPublic {
		return ch, message.ChannelForbiddenErr
	}
	exists, err := message.IsChannelMember(ctx, db, ch.Id, user.UserId)
	if err != nil {
		return ch, err
	}
	if exists {
		return ch, nil
	}
	m := message.ChannelMember{
		ChannelId: ch.Id,
		UserId:    user.UserId,
		Role:      message.ChannelRoleSubscriber,
	}
	if err := message.CreateChannelMember(ctx, db, &m); err != nil {
		return ch, err
	}

	ids, _ := message.ListChannelMemberUserIds(ctx, db, ch.Id)
	if rdb != nil {
		_ = PublishChannelAdded(ctx, rdb, []int{user.UserId}, ChannelToPayload(ch, len(ids)))
		if ch.E2EEEnabled {
			_ = PublishChannelMembersChanged(ctx, rdb, ids, ChannelMembersChangedPayload{
				ChannelUuid:  ch.ChannelUuid.String(),
				Title:        ch.Title,
				AddedUserIds: []int{user.UserId},
				MemberCount:  len(ids),
				ActorUserId:  user.UserId,
				E2EEEnabled:  ch.E2EEEnabled,
			})
		}
	}
	return ch, nil
}

func UnsubscribeFromChannel(ctx context.Context, db *gorm.DB, rdb *redis.Client, user message.UserRelation, channelUuid uuid.UUID) error {
	ch, err := message.GetChannelByUuid(ctx, db, channelUuid)
	if err != nil {
		return err
	}
	member, err := message.GetChannelMember(ctx, db, ch.Id, user.UserId)
	if err != nil {
		return err
	}
	if member.Role == message.ChannelRoleOwner {
		return fmt.Errorf("owner cannot unsubscribe; transfer ownership or delete channel")
	}
	if err := message.DeleteChannelMember(ctx, db, ch.Id, user.UserId); err != nil {
		return err
	}
	ids, _ := message.ListChannelMemberUserIds(ctx, db, ch.Id)
	if rdb != nil && ch.E2EEEnabled {
		_ = PublishChannelMembersChanged(ctx, rdb, ids, ChannelMembersChangedPayload{
			ChannelUuid:    ch.ChannelUuid.String(),
			Title:          ch.Title,
			RemovedUserIds: []int{user.UserId},
			MemberCount:    len(ids),
			ActorUserId:    user.UserId,
			E2EEEnabled:    ch.E2EEEnabled,
		})
	}
	return nil
}

// DeleteChannelMessage удаляет пост канала. Может отправитель или owner/admin.
// Рассылает type=delete с deleted_message_id всем участникам.
func UpdateChannelMessage(ctx context.Context, db *gorm.DB, rdb *redis.Client, actor message.UserRelation, channelUuid uuid.UUID, messageID int, text string) error {
	ch, err := message.GetChannelByUuid(ctx, db, channelUuid)
	if err != nil {
		return err
	}
	member, err := message.IsChannelMember(ctx, db, ch.Id, actor.UserId)
	if err != nil {
		return err
	}
	if !member {
		return message.ChannelForbiddenErr
	}
	link, err := message.GetChannelMsgByMessageId(ctx, db, messageID)
	if err != nil {
		return err
	}
	if link.ChannelId != ch.ChannelUuid {
		return message.ChannelMsgNotFoundErr
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
	memberIds, err := message.ListChannelMemberUserIds(ctx, db, ch.Id)
	if err != nil {
		return err
	}
	protoMsg := &messenger.Message{Id: int64(messageID), SenderId: int64(actor.UserId), Text: updated.Text, CreatedAt: timestamppb.New(updated.CreatedAt.Time)}
	eventData, err := json.Marshal(ChannelMessageEvent{Type: grpc.TypeNewMessage, Message: protoMsg, ChannelId: ch.ChannelUuid.String(), Edited: true})
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

func DeleteChannelMessage(
	ctx context.Context,
	db *gorm.DB,
	rdb *redis.Client,
	actor message.UserRelation,
	channelUuid uuid.UUID,
	messageID int,
) error {
	ch, err := message.GetChannelByUuid(ctx, db, channelUuid)
	if err != nil {
		return err
	}

	link, err := message.GetChannelMsgByMessageId(ctx, db, messageID)
	if err != nil {
		return err
	}
	if link.ChannelId != ch.ChannelUuid {
		return message.ChannelMsgNotFoundErr
	}

	if link.SenderId != actor.UserId {
		can, err := message.CanPostToChannel(ctx, db, ch.Id, actor.UserId)
		if err != nil {
			return err
		}
		if !can {
			return message.ChannelPostForbiddenErr
		}
	}

	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := message.DeleteMessageById(ctx, tx, messageID); err != nil {
			return err
		}
		if err := message.DeleteChannelMsgById(ctx, tx, link.Id); err != nil {
			//TODO: реализовать
		}
		d := message.Deleted{
			MessageId: messageID,
			UserId:    actor.UserId,
		}
		_ = message.CreateDeleted(ctx, tx, &d)
		return nil
	})
	if err != nil {
		return err
	}

	if rdb != nil {
		_ = PublishChannelMessageDeleted(ctx, db, rdb, ch, int64(messageID))
	}
	return nil
}

// PublishChannelMessageDeleted — fan-out delete события подписчикам канала.
func PublishChannelMessageDeleted(ctx context.Context, db *gorm.DB, rdb *redis.Client, ch message.Channel, messageID int64) error {
	if rdb == nil {
		return nil
	}
	memberIds, err := message.ListChannelMemberUserIds(ctx, db, ch.Id)
	if err != nil {
		return err
	}
	eventData, err := json.Marshal(ChannelMessageEvent{
		Type:             grpc.TypeDelete,
		ChannelId:        ch.ChannelUuid.String(),
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
