package message

import (
	"chickchirick-messages/internal/enum/grpc"
	"chickchirick-messages/internal/gen/messenger" // Путь к сгенерированному коду
	"chickchirick-messages/internal/model/message"
	"chickchirick-messages/internal/service"
	"encoding/json"
	"fmt"

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

type RedisMessageEvent struct {
	Type             grpc.MessageType   `json:"type"`
	Message          *messenger.Message `json:"message,omitempty"`
	DeletedMessageID int64              `json:"deleted_message_id,omitempty"`
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
				grpcEvent.Event = &messenger.MessageEvent_Message{Message: event.Message}
			} else if event.Type == grpc.TypeDelete {
				grpcEvent.Event = &messenger.MessageEvent_DeletedMessageId{DeletedMessageId: event.DeletedMessageID}
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

			msg, err := service.CreateMessage(ctx, c.db, req, userRelation)
			if err != nil {
				errChan <- err
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
