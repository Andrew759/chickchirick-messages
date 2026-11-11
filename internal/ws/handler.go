package ws

import (
	"context"
	"log"
	"net/http"
	"time"

	"chickchirick-messages/internal/gen/messenger"
	"chickchirick-messages/internal/model/message"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// IncomingMessage — сообщение от фронтенда (Vue → WS → gRPC)
type IncomingMessage struct {
	Type        string `json:"type,omitempty"`
	RecipientId int64  `json:"recipientId"`
	GroupId     string `json:"groupId,omitempty"` // UUID группового чата
	Text        string `json:"text,omitempty"`
	Typing      *bool  `json:"typing,omitempty"`
}

// OutgoingEvent — JSON, который понимает фронтенд
type OutgoingEvent struct {
	Message          *OutgoingMessage `json:"message,omitempty"`
	DeletedMessageId int64            `json:"deletedMessageId,omitempty"`
	Typing           *TypingEvent     `json:"typing,omitempty"`
}

type TypingEvent struct {
	SenderId    int64  `json:"senderId"`
	RecipientId int64  `json:"recipientId"`
	GroupId     string `json:"groupId,omitempty"`
	Typing      bool   `json:"typing"`
}

type OutgoingMessage struct {
	Id          int64  `json:"id"`
	SenderId    int64  `json:"senderId"`
	RecipientId int64  `json:"recipientId"`
	GroupId     string `json:"groupId,omitempty"`
	Text        string `json:"text"`
	CreatedAt   string `json:"createdAt,omitempty"`
}

// HandleWS grpcAddr — gRPC messenger; db опционален (для groupId lookup).
func HandleWS(grpcAddr string, db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wsConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("ws upgrade error:", err)
			return
		}
		defer wsConn.Close()

		cookie, err := r.Cookie("access_token")
		if err != nil || cookie.Value == "" {
			log.Println("token missing: access_token cookie required")
			_ = wsConn.WriteJSON(map[string]string{"error": "unauthorized"})
			return
		}

		conn, err := grpc.NewClient(
			grpcAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			log.Println("grpc dial error:", err)
			return
		}
		defer conn.Close()

		client := messenger.NewMessengerServiceClient(conn)

		rawCookie := r.Header.Get("Cookie")
		if rawCookie == "" {
			rawCookie = "access_token=" + cookie.Value
		}
		md := metadata.New(map[string]string{
			"cookie": rawCookie,
		})
		ctx := metadata.NewOutgoingContext(r.Context(), md)

		stream, err := client.MessageStream(ctx)
		if err != nil {
			log.Println("stream error:", err)
			_ = wsConn.WriteJSON(map[string]string{"error": "stream failed: " + err.Error()})
			return
		}

		go func() {
			for {
				var msg IncomingMessage
				if err := wsConn.ReadJSON(&msg); err != nil {
					log.Println("read ws error:", err)
					return
				}

				// Групповые сообщения уходят через REST /group-chats/:uuid/messages
				if msg.GroupId != "" {
					log.Println("ws: group messages should use REST API, groupId=", msg.GroupId)
					continue
				}

				if msg.RecipientId == 0 {
					log.Println("invalid incoming message:", msg)
					continue
				}

				req := &messenger.SendMessageRequest{
					RecipientId: msg.RecipientId,
					Text:        msg.Text,
				}

				if msg.Type == "typing" {
					if msg.Typing != nil {
						active := *msg.Typing
						req.Typing = &active
					} else {
						active := msg.Text != "0"
						req.Typing = &active
					}
					req.Text = ""
				}

				err := stream.Send(req)
				if err != nil {
					log.Println("stream send error:", err)
					return
				}
			}
		}()

		marshaler := protojson.MarshalOptions{
			UseProtoNames:   false,
			EmitUnpopulated: false,
		}

		for {
			res, err := stream.Recv()
			if err != nil {
				log.Println("stream recv error:", err)
				return
			}

			out := OutgoingEvent{}
			switch e := res.Event.(type) {
			case *messenger.MessageEvent_Message:
				m := e.Message
				createdAt := time.Now().UTC().Format(time.RFC3339Nano)
				if m.CreatedAt != nil {
					createdAt = m.CreatedAt.AsTime().UTC().Format(time.RFC3339Nano)
				}
				groupId := ""
				if m.RecipientId == 0 && db != nil && m.Id > 0 {
					if link, err := message.GetGroupByMessageId(context.Background(), db, int(m.Id)); err == nil {
						groupId = link.GroupId.String()
						if m.SenderId == 0 && link.SenderId > 0 {
							m.SenderId = int64(link.SenderId)
						}
					}
				}
				out.Message = &OutgoingMessage{
					Id:          m.Id,
					SenderId:    m.SenderId,
					RecipientId: m.RecipientId,
					GroupId:     groupId,
					Text:        m.Text,
					CreatedAt:   createdAt,
				}
			case *messenger.MessageEvent_DeletedMessageId:
				out.DeletedMessageId = e.DeletedMessageId
			case *messenger.MessageEvent_Typing:
				if e.Typing != nil {
					out.Typing = &TypingEvent{
						SenderId:    e.Typing.SenderId,
						RecipientId: e.Typing.RecipientId,
						Typing:      e.Typing.Typing,
					}
				} else {
					continue
				}
			default:
				b, err := marshaler.Marshal(res)
				if err != nil {
					log.Println("proto marshal error:", err)
					continue
				}
				if err := wsConn.WriteMessage(websocket.TextMessage, b); err != nil {
					log.Println("ws write error:", err)
					return
				}
				continue
			}

			if err := wsConn.WriteJSON(out); err != nil {
				log.Println("ws write error:", err)
				return
			}
		}
	}
}
