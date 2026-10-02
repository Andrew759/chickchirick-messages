package grpc

import "encoding/json"

type MessageType int

const (
	TypeNewMessage MessageType = iota + 1
	TypeDelete
	TypeTyping
	TypeUnknown
)

// Строковые соответствия для JSON сериализации
const (
	strNewMessage = "new_message"
	strDelete     = "delete"
	StrTyping     = "typing"
)

// MarshalJSON преобразует enum в строку для Redis/JSON
func (t MessageType) MarshalJSON() ([]byte, error) {
	switch t {
	case TypeNewMessage:
		return json.Marshal(strNewMessage)
	case TypeDelete:
		return json.Marshal(strDelete)
	case TypeTyping:
		return json.Marshal(StrTyping)
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
	case StrTyping:
		*t = TypeTyping
	default:
		*t = TypeUnknown
	}
	return nil
}
