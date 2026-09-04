package grpc

import "encoding/json"

type MessageType int

const (
	TypeNewMessage MessageType = iota + 1
	TypeDelete
	TypeUnknown
)

// Строковые соответствия для JSON сериализации
const (
	strNewMessage = "new_message"
	strDelete     = "delete"
)

// MarshalJSON преобразует enum в строку для Redis/JSON
func (t MessageType) MarshalJSON() ([]byte, error) {
	switch t {
	case TypeNewMessage:
		return json.Marshal(strNewMessage)
	case TypeDelete:
		return json.Marshal(strDelete)
	default:
		return json.Marshal("")
	}
}

// UnmarshalJSON парсит строку из Redis/JSON в enum
func (t *MessageType) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	switch s {
	case strNewMessage:
		*t = TypeNewMessage
	case strDelete:
		*t = TypeDelete
	default:
		*t = TypeUnknown
	}
	return nil
}
