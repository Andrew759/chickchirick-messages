package grpc

import "encoding/json"

type MessageType int

const (
	TypeNewMessage MessageType = iota + 1
	TypeDelete
	TypeTyping
	TypeGroupAdded
	TypeGroupMembersChanged
	TypeChannelAdded
	TypeChannelMembersChanged
	TypeUnknown
)

const (
	strNewMessage            = "new_message"
	strDelete                = "delete"
	StrTyping                = "typing"
	strGroupAdded            = "group_added"
	strGroupMembersChanged   = "group_members_changed"
	strChannelAdded          = "channel_added"
	strChannelMembersChanged = "channel_members_changed"
)

func (t MessageType) MarshalJSON() ([]byte, error) {
	switch t {
	case TypeNewMessage:
		return json.Marshal(strNewMessage)
	case TypeDelete:
		return json.Marshal(strDelete)
	case TypeTyping:
		return json.Marshal(StrTyping)
	case TypeGroupAdded:
		return json.Marshal(strGroupAdded)
	case TypeGroupMembersChanged:
		return json.Marshal(strGroupMembersChanged)
	case TypeChannelAdded:
		return json.Marshal(strChannelAdded)
	case TypeChannelMembersChanged:
		return json.Marshal(strChannelMembersChanged)
	default:
		return json.Marshal("")
	}
}

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
	case strGroupAdded:
		*t = TypeGroupAdded
	case strGroupMembersChanged:
		*t = TypeGroupMembersChanged
	case strChannelAdded:
		*t = TypeChannelAdded
	case strChannelMembersChanged:
		*t = TypeChannelMembersChanged
	default:
		*t = TypeUnknown
	}
	return nil
}
