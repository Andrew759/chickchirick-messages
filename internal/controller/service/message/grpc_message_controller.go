package message

import (
	"chickchirick-messages/internal/enum/grpc"
	"chickchirick-messages/internal/gen/messenger" // Путь к сгенерированному коду
	"chickchirick-messages/internal/model/message"
	"chickchirick-messages/internal/service"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type Controller struct {
	messenger.UnimplementedMessengerServiceServer
	db    *gorm.DB
	redis *redis.Client
}

func NewController(db *gorm.DB, redis *redis.Client) *Controller {
	return &Controller{
		db:    db,
		redis: redis,
	}
}

type RedisTypingEvent struct {
	SenderID    int64  `json:"sender_id"`
	RecipientID int64  `json:"recipient_id"`
	GroupUuid   string `json:"group_uuid,omitempty"`
	Typing      bool   `json:"typing"`
}

type RedisGroupPayload struct {
	GroupUuid      string  `json:"group_uuid"`
	Title          string  `json:"title"`
	AvatarFileUuid *string `json:"avatar_file_uuid,omitempty"`
	CreatedBy      int     `json:"created_by"`
	MemberCount    int     `json:"member_count"`
}

type RedisMembersChangedPayload struct {
	GroupUuid      string `json:"group_uuid"`
	Title          string `json:"title,omitempty"`
	AddedUserIds   []int  `json:"added_user_ids,omitempty"`
	RemovedUserIds []int  `json:"removed_user_ids,omitempty"`
	MemberCount    int    `json:"member_count"`
	ActorUserId    int    `json:"actor_user_id"`
}

type RedisGroupTypingPayload struct {
	GroupUuid string `json:"group_uuid"`
	SenderID  int64  `json:"sender_id"`
	Typing    bool   `json:"typing"`
}

type RedisChannelPayload struct {
	ChannelUuid    string  `json:"channel_uuid"`
	Title          string  `json:"title"`
	Description    string  `json:"description,omitempty"`
	AvatarFileUuid *string `json:"avatar_file_uuid,omitempty"`
	IsPublic       bool    `json:"is_public"`
	E2EEEnabled    bool    `json:"e2ee_enabled"`
	CreatedBy      int     `json:"created_by"`
	MemberCount    int     `json:"member_count"`
}

type RedisChannelMembersChangedPayload struct {
	ChannelUuid    string `json:"channel_uuid"`
	Title          string `json:"title,omitempty"`
	AddedUserIds   []int  `json:"added_user_ids,omitempty"`
	RemovedUserIds []int  `json:"removed_user_ids,omitempty"`
	MemberCount    int    `json:"member_count"`
	ActorUserId    int    `json:"actor_user_id"`
	E2EEEnabled    bool   `json:"e2ee_enabled"`
}

type RedisChannelTypingPayload struct {
	ChannelUuid string `json:"channel_uuid"`
	SenderID    int64  `json:"sender_id"`
	Typing      bool   `json:"typing"`
}

type RedisMessageEvent struct {
	Type             grpc.MessageType                   `json:"type"`
	Message          *messenger.Message                 `json:"message,omitempty"`
	GroupId          string                             `json:"group_id,omitempty"`
	ChannelId        string                             `json:"channel_id,omitempty"`
	Group            *RedisGroupPayload                 `json:"group,omitempty"`
	Channel          *RedisChannelPayload               `json:"channel,omitempty"`
	MembersChanged   *RedisMembersChangedPayload        `json:"members_changed,omitempty"`
	ChannelMembers   *RedisChannelMembersChangedPayload `json:"channel_members_changed,omitempty"`
	DeletedMessageID int64                              `json:"deleted_message_id,omitempty"`
	Edited           bool                               `json:"edited,omitempty"`
	Typing           *RedisTypingEvent                  `json:"typing,omitempty"`
	// service.PublishGroupTyping / PublishChannelTyping — typing payload
}

func (c *Controller) MessageStream(stream messenger.MessengerService_MessageStreamServer) error {
	ctx := stream.Context()

	userUuid := ctx.Value("user_uuid").(string)
	userRelation, err := message.GetUserRelationByUuid(ctx, c.db, userUuid)
	if err != nil {
		return err
	}

	//Подписываемся на личный топик пользователя в Redis
	pubSub := c.redis.Subscribe(ctx, fmt.Sprintf("user_events_%d", userRelation.UserId))
	defer pubSub.Close()

	errChan := make(chan error, 2)

	//Прослушка Redis и отправка события в Stream (Outbound)
	go func() {
		for {
			msg, err := pubSub.ReceiveMessage(ctx)
			if err != nil {
				errChan <- err
				return
			}

			var event RedisMessageEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				continue
			}

			//Формирование gRPC ответа на основе типа события
			grpcEvent := &messenger.MessageEvent{}
			if event.Type == grpc.TypeNewMessage {
				msg := event.Message
				// Явный group_id вместо recipient_id=0: временный транспортный префикс, WS снимает.
				if msg != nil && event.Edited {
					msg = &messenger.Message{
						Id: msg.Id, SenderId: msg.SenderId, RecipientId: msg.RecipientId,
						Text: "__EDITED__" + msg.Text, CreatedAt: msg.CreatedAt,
						RespondMessageId: msg.RespondMessageId,
					}
				}
				if msg != nil && event.GroupId != "" {
					cloned := &messenger.Message{
						Id:               msg.Id,
						SenderId:         msg.SenderId,
						RecipientId:      msg.RecipientId,
						Text:             "__GID__:" + event.GroupId + "\n" + msg.Text,
						CreatedAt:        msg.CreatedAt,
						RespondMessageId: msg.RespondMessageId,
					}
					msg = cloned
				}
				if msg != nil && event.ChannelId != "" {
					cloned := &messenger.Message{
						Id:               msg.Id,
						SenderId:         msg.SenderId,
						RecipientId:      msg.RecipientId,
						Text:             "__CID__:" + event.ChannelId + "\n" + msg.Text,
						CreatedAt:        msg.CreatedAt,
						RespondMessageId: msg.RespondMessageId,
					}
					msg = cloned
				}
				grpcEvent.Event = &messenger.MessageEvent_Message{Message: msg}
			} else if event.Type == grpc.TypeDelete {
				grpcEvent.Event = &messenger.MessageEvent_DeletedMessageId{DeletedMessageId: event.DeletedMessageID}
			} else if event.Type == grpc.TypeTyping && event.Typing != nil {
				// Канальный typing
				if event.ChannelId != "" || (event.Typing != nil && event.Typing.GroupUuid == "" && event.GroupId == "" && event.ChannelId != "") {
					// handled below if ChannelId set on event
				}
				if event.ChannelId != "" {
					raw, _ := json.Marshal(map[string]interface{}{
						"channel_uuid": event.ChannelId,
						"sender_id":    event.Typing.SenderID,
						"typing":       event.Typing.Typing,
					})
					// Prefer explicit channel typing payload from service
					grpcEvent.Event = &messenger.MessageEvent_Message{
						Message: &messenger.Message{
							SenderId:  event.Typing.SenderID,
							Text:      "__CHANNEL_TYPING__" + string(raw),
							CreatedAt: timestamppb.Now(),
						},
					}
				} else if event.Typing.GroupUuid != "" || event.GroupId != "" {
					// Групповой typing: прокидываем через Message text (в proto TypingEvent нет group_id)
					gid := event.Typing.GroupUuid
					if gid == "" {
						gid = event.GroupId
					}
					raw, _ := json.Marshal(map[string]interface{}{
						"group_uuid": gid,
						"sender_id":  event.Typing.SenderID,
						"typing":     event.Typing.Typing,
					})
					grpcEvent.Event = &messenger.MessageEvent_Message{
						Message: &messenger.Message{
							SenderId:  event.Typing.SenderID,
							Text:      "__GROUP_TYPING__" + string(raw),
							CreatedAt: timestamppb.Now(),
						},
					}
				} else {
					grpcEvent.Event = &messenger.MessageEvent_Typing{
						Typing: &messenger.TypingEvent{
							SenderId:    event.Typing.SenderID,
							RecipientId: event.Typing.RecipientID,
							Typing:      event.Typing.Typing,
						},
					}
				}
			} else if event.Type == grpc.TypeGroupAdded && event.Group != nil {
				raw, _ := json.Marshal(event.Group)
				grpcEvent.Event = &messenger.MessageEvent_Message{
					Message: &messenger.Message{
						Text:      "__GROUP_ADDED__" + string(raw),
						CreatedAt: timestamppb.Now(),
					},
				}
			} else if event.Type == grpc.TypeGroupMembersChanged && event.MembersChanged != nil {
				raw, _ := json.Marshal(event.MembersChanged)
				grpcEvent.Event = &messenger.MessageEvent_Message{
					Message: &messenger.Message{
						SenderId:  int64(event.MembersChanged.ActorUserId),
						Text:      "__GROUP_MEMBERS_CHANGED__" + string(raw),
						CreatedAt: timestamppb.Now(),
					},
				}
			} else if event.Type == grpc.TypeChannelAdded && event.Channel != nil {
				raw, _ := json.Marshal(event.Channel)
				grpcEvent.Event = &messenger.MessageEvent_Message{
					Message: &messenger.Message{
						Text:      "__CHANNEL_ADDED__" + string(raw),
						CreatedAt: timestamppb.Now(),
					},
				}
			} else if event.Type == grpc.TypeChannelMembersChanged && event.ChannelMembers != nil {
				raw, _ := json.Marshal(event.ChannelMembers)
				grpcEvent.Event = &messenger.MessageEvent_Message{
					Message: &messenger.Message{
						SenderId:  int64(event.ChannelMembers.ActorUserId),
						Text:      "__CHANNEL_MEMBERS_CHANGED__" + string(raw),
						CreatedAt: timestamppb.Now(),
					},
				}
			} else if event.Type == grpc.TypeChannelMembersChanged && event.MembersChanged != nil {
				// fallback: service may reuse members_changed key shape with channel_uuid inside
				raw, _ := json.Marshal(event.MembersChanged)
				grpcEvent.Event = &messenger.MessageEvent_Message{
					Message: &messenger.Message{
						SenderId:  int64(event.MembersChanged.ActorUserId),
						Text:      "__CHANNEL_MEMBERS_CHANGED__" + string(raw),
						CreatedAt: timestamppb.Now(),
					},
				}
			} else {
				continue
			}

			if err := stream.Send(grpcEvent); err != nil {
				errChan <- err
				return
			}
		}
	}()

	//Прослушка Stream и публикация в Redis + БД (Inbound)
	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				errChan <- err
				return
			}

			if req.Typing != nil {
				if strings.HasPrefix(req.GetText(), "__CHANNEL_TYPING_CMD__") {
					cidStr := strings.TrimPrefix(req.GetText(), "__CHANNEL_TYPING_CMD__")
					cu, err := uuid.Parse(cidStr)
					if err != nil {
						continue
					}
					_ = service.PublishChannelTyping(ctx, c.db, c.redis, cu, userRelation.UserId, req.GetTyping())
					continue
				}
				if strings.HasPrefix(req.GetText(), "__GROUP_TYPING_CMD__") {
					gidStr := strings.TrimPrefix(req.GetText(), "__GROUP_TYPING_CMD__")
					gu, err := uuid.Parse(gidStr)
					if err != nil {
						continue
					}
					_ = service.PublishGroupTyping(ctx, c.db, c.redis, gu, userRelation.UserId, req.GetTyping())
					continue
				}
				eventData, err := json.Marshal(RedisMessageEvent{
					Type: grpc.TypeTyping,
					Typing: &RedisTypingEvent{
						SenderID:    int64(userRelation.UserId),
						RecipientID: req.RecipientId,
						Typing:      req.GetTyping(),
					},
				})
				if err != nil {
					errChan <- err
					return
				}
				if err := c.redis.Publish(ctx, fmt.Sprintf("user_events_%d", req.RecipientId), eventData).Err(); err != nil {
					errChan <- err
					return
				}
				continue
			}

			msg, err := service.CreateMessage(ctx, c.db, req, userRelation)
			if err != nil {
				errChan <- err
				return
			}

			//Формирование объекта для рассылки
			protoMsg := &messenger.Message{
				Id:               int64(msg.Id),
				SenderId:         int64(userRelation.UserId),
				RecipientId:      req.RecipientId,
				Text:             req.Text,
				CreatedAt:        timestamppb.Now(),
				RespondMessageId: req.RespondMessageId,
			}

			eventData, _ := json.Marshal(RedisMessageEvent{
				Type:    grpc.TypeNewMessage,
				Message: protoMsg,
			})

			//Публикация получателю
			c.redis.Publish(ctx, fmt.Sprintf("user_events_%d", req.RecipientId), eventData)
			//И себе (чтобы сообщение появилось на других устройствах)
			c.redis.Publish(ctx, fmt.Sprintf("user_events_%d", userRelation.UserId), eventData)
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}
