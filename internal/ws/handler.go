package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
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
	GroupId     string `json:"groupId,omitempty"`   // UUID группового чата
	ChannelId   string `json:"channelId,omitempty"` // UUID канала
	Text        string `json:"text,omitempty"`
	Typing      *bool  `json:"typing,omitempty"`
}

// OutgoingEvent — JSON, который понимает фронтенд
type OutgoingEvent struct {
	Message               *OutgoingMessage            `json:"message,omitempty"`
	DeletedMessageId      int64                       `json:"deletedMessageId,omitempty"`
	Typing                *TypingEvent                `json:"typing,omitempty"`
	GroupAdded            *GroupAddedEvent            `json:"groupAdded,omitempty"`
	GroupMembersChanged   *GroupMembersChangedEvent   `json:"groupMembersChanged,omitempty"`
	ChannelAdded          *ChannelAddedEvent          `json:"channelAdded,omitempty"`
	ChannelMembersChanged *ChannelMembersChangedEvent `json:"channelMembersChanged,omitempty"`
}

// GroupAddedEvent — вас добавили в группу (realtime)
type GroupAddedEvent struct {
	GroupUuid      string  `json:"group_uuid"`
	Title          string  `json:"title"`
	AvatarFileUuid *string `json:"avatar_file_uuid,omitempty"`
	CreatedBy      int     `json:"created_by"`
	MemberCount    int     `json:"member_count"`
}

// GroupMembersChangedEvent — состав группы изменился (E2EE re-key)
type GroupMembersChangedEvent struct {
	GroupUuid      string `json:"group_uuid"`
	Title          string `json:"title,omitempty"`
	AddedUserIds   []int  `json:"added_user_ids,omitempty"`
	RemovedUserIds []int  `json:"removed_user_ids,omitempty"`
	MemberCount    int    `json:"member_count"`
	ActorUserId    int    `json:"actor_user_id"`
}

type TypingEvent struct {
	SenderId    int64  `json:"senderId"`
	RecipientId int64  `json:"recipientId"`
	GroupId     string `json:"groupId,omitempty"`
	ChannelId   string `json:"channelId,omitempty"`
	Typing      bool   `json:"typing"`
}

type ChannelAddedEvent struct {
	ChannelUuid    string  `json:"channel_uuid"`
	Title          string  `json:"title"`
	Description    string  `json:"description,omitempty"`
	AvatarFileUuid *string `json:"avatar_file_uuid,omitempty"`
	IsPublic       bool    `json:"is_public"`
	E2EEEnabled    bool    `json:"e2ee_enabled"`
	CreatedBy      int     `json:"created_by"`
	MemberCount    int     `json:"member_count"`
}

type ChannelMembersChangedEvent struct {
	ChannelUuid    string `json:"channel_uuid"`
	Title          string `json:"title,omitempty"`
	AddedUserIds   []int  `json:"added_user_ids,omitempty"`
	RemovedUserIds []int  `json:"removed_user_ids,omitempty"`
	MemberCount    int    `json:"member_count"`
	ActorUserId    int    `json:"actor_user_id"`
	E2EEEnabled    bool   `json:"e2ee_enabled"`
}

type OutgoingMessage struct {
	Id          int64  `json:"id"`
	SenderId    int64  `json:"senderId"`
	RecipientId int64  `json:"recipientId"`
	GroupId     string `json:"groupId,omitempty"`
	ChannelId   string `json:"channelId,omitempty"`
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

				if msg.ChannelId != "" {
					isTyping := msg.Type == "typing" || msg.Typing != nil
					if !isTyping {
						log.Println("ws: channel messages should use REST API, channelId=", msg.ChannelId)
						continue
					}
					active := false
					if msg.Typing != nil {
						active = *msg.Typing
					} else {
						active = msg.Text != "0" && msg.Text != ""
					}
					req := &messenger.SendMessageRequest{
						Text:   "__CHANNEL_TYPING_CMD__" + msg.ChannelId,
						Typing: &active,
					}
					if err := stream.Send(req); err != nil {
						log.Println("stream send channel typing error:", err)
						return
					}
					continue
				}

				if msg.GroupId != "" {
					isTyping := msg.Type == "typing" || msg.Typing != nil
					if !isTyping {
						log.Println("ws: group messages should use REST API, groupId=", msg.GroupId)
						continue
					}
					active := false
					if msg.Typing != nil {
						active = *msg.Typing
					} else {
						active = msg.Text != "0" && msg.Text != ""
					}
					req := &messenger.SendMessageRequest{
						Text: "__GROUP_TYPING_CMD__" + msg.GroupId,
						// Групповой typing: Text = __GROUP_TYPING_CMD__<uuid>
						Typing: &active,
					}
					if err := stream.Send(req); err != nil {
						log.Println("stream send group typing error:", err)
						return
					}
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
				// realtime: добавление в группу (транспорт через text-prefix)
				if strings.HasPrefix(m.Text, "__GROUP_ADDED__") {
					var g GroupAddedEvent
					if err := json.Unmarshal([]byte(strings.TrimPrefix(m.Text, "__GROUP_ADDED__")), &g); err == nil && g.GroupUuid != "" {
						out.GroupAdded = &g
						break
					}
				}
				if strings.HasPrefix(m.Text, "__GROUP_MEMBERS_CHANGED__") {
					var g GroupMembersChangedEvent
					if err := json.Unmarshal([]byte(strings.TrimPrefix(m.Text, "__GROUP_MEMBERS_CHANGED__")), &g); err == nil && g.GroupUuid != "" {
						out.GroupMembersChanged = &g
						break
					}
				}
				if strings.HasPrefix(m.Text, "__GROUP_TYPING__") {
					var gt struct {
						GroupUuid string `json:"group_uuid"`
						SenderID  int64  `json:"sender_id"`
						Typing    bool   `json:"typing"`
					}
					if err := json.Unmarshal([]byte(strings.TrimPrefix(m.Text, "__GROUP_TYPING__")), &gt); err == nil && gt.GroupUuid != "" {
						out.Typing = &TypingEvent{
							SenderId: gt.SenderID,
							GroupId:  gt.GroupUuid,
							Typing:   gt.Typing,
						}
						break
					}
				}
				if strings.HasPrefix(m.Text, "__CHANNEL_ADDED__") {
					var ch ChannelAddedEvent
					if err := json.Unmarshal([]byte(strings.TrimPrefix(m.Text, "__CHANNEL_ADDED__")), &ch); err == nil && ch.ChannelUuid != "" {
						out.ChannelAdded = &ch
						break
					}
				}
				if strings.HasPrefix(m.Text, "__CHANNEL_MEMBERS_CHANGED__") {
					var ch ChannelMembersChangedEvent
					if err := json.Unmarshal([]byte(strings.TrimPrefix(m.Text, "__CHANNEL_MEMBERS_CHANGED__")), &ch); err == nil && ch.ChannelUuid != "" {
						out.ChannelMembersChanged = &ch
						break
					}
				}
				if strings.HasPrefix(m.Text, "__CHANNEL_TYPING__") {
					var ct struct {
						ChannelUuid string `json:"channel_uuid"`
						SenderID    int64  `json:"sender_id"`
						Typing      bool   `json:"typing"`
					}
					if err := json.Unmarshal([]byte(strings.TrimPrefix(m.Text, "__CHANNEL_TYPING__")), &ct); err == nil && ct.ChannelUuid != "" {
						out.Typing = &TypingEvent{
							SenderId:  ct.SenderID,
							ChannelId: ct.ChannelUuid,
							Typing:    ct.Typing,
						}
						break
					}
				}
				createdAt := time.Now().UTC().Format(time.RFC3339Nano)
				if m.CreatedAt != nil {
					createdAt = m.CreatedAt.AsTime().UTC().Format(time.RFC3339Nano)
				}
				text := m.Text
				groupId := ""
				channelId := ""
				// Явный group_id из MessageStream: "__GID__:<uuid>\n" + body
				if strings.HasPrefix(text, "__GID__:") {
					rest := strings.TrimPrefix(text, "__GID__:")
					if i := strings.IndexByte(rest, '\n'); i >= 0 {
						groupId = rest[:i]
						text = rest[i+1:]
					}
				}
				// Явный channel_id: "__CID__:<uuid>\n" + body
				if strings.HasPrefix(text, "__CID__:") {
					rest := strings.TrimPrefix(text, "__CID__:")
					if i := strings.IndexByte(rest, '\n'); i >= 0 {
						channelId = rest[:i]
						text = rest[i+1:]
					}
				}
				// Fallback: связь message → group в БД
				if groupId == "" && db != nil && m.Id > 0 {
					if link, err := message.GetGroupByMessageId(context.Background(), db, int(m.Id)); err == nil {
						groupId = link.GroupId.String()
						if m.SenderId == 0 && link.SenderId > 0 {
							m.SenderId = int64(link.SenderId)
						}
					}
				}
				// Fallback: связь message → channel
				if channelId == "" && db != nil && m.Id > 0 {
					if link, err := message.GetChannelMsgByMessageId(context.Background(), db, int(m.Id)); err == nil {
						channelId = link.ChannelId.String()
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
					ChannelId:   channelId,
					Text:        text,
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
