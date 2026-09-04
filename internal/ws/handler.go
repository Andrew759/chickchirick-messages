package ws

import (
	"log"
	"net/http"

	"chickchirick-messages/internal/gen/messenger"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type IncomingMessage struct {
	RecipientId int64  `json:"recipientId"`
	Text        string `json:"text"`
}

type OutgoingEvent struct {
	Message          *OutgoingMessage `json:"message,omitempty"`
	DeletedMessageId int64            `json:"deletedMessageId,omitempty"`
}

type OutgoingMessage struct {
	Id          int64  `json:"id"`
	SenderId    int64  `json:"senderId"`
	RecipientId int64  `json:"recipientId"`
	Text        string `json:"text"`
	CreatedAt   string `json:"createdAt,omitempty"`
}

func HandleWS(grpcAddr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wsConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("ws upgrade error:", err)
			return
		}
		defer wsConn.Close()

		var tokenStr string
		if cookie, err := r.Cookie("access_token"); err == nil && cookie.Value != "" {
			tokenStr = cookie.Value
		}

		if tokenStr == "" {
			log.Println("token missing")
			_ = wsConn.WriteJSON(map[string]string{"error": "unauthorized"})
			return
		}

		//TODO: для прода должны быть secure credentials
		//Подключаемся к своему же gRPC-серверу (обязательны transport credentials)
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
			rawCookie = "access_token=" + tokenStr
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

		//Vue → gRPC
		go func() {
			for {
				var msg IncomingMessage
				if err := wsConn.ReadJSON(&msg); err != nil {
					log.Println("read ws error:", err)
					return
				}

				if msg.RecipientId == 0 || msg.Text == "" {
					log.Println("invalid incoming message:", msg)
					continue
				}

				err := stream.Send(&messenger.SendMessageRequest{
					RecipientId: msg.RecipientId,
					Text:        msg.Text,
				})
				if err != nil {
					log.Println("stream send error:", err)
					return
				}
			}
		}()

		//gRPC → Vue (нормальный JSON, а не сырой protobuf)
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
				out.Message = &OutgoingMessage{
					Id:          m.Id,
					SenderId:    m.SenderId,
					RecipientId: m.RecipientId,
					Text:        m.Text,
				}
				if m.CreatedAt != nil {
					out.Message.CreatedAt = m.CreatedAt.AsTime().Format("2006-01-02T15:04:05Z07:00")
				}
			case *messenger.MessageEvent_DeletedMessageId:
				out.DeletedMessageId = e.DeletedMessageId
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
