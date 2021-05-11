package types

import (
	"encoding/json"
	"time"
)

// Message wraps serialized messages for Git and ROS transports
type Message struct {
	Timestamp   time.Time `json:"timestamp"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	ID          string    `json:"id"`
	MessageType string    `json:"message_type"`
	Message     string    `json:"message"`
}

// MessageOut wraps typed structs
type MessageOut struct {
	Timestamp   time.Time
	From        string
	To          string
	ID          string
	MessageType string
	Message     interface{}
}

// CreateMessage convert MessageOut into a Message using JSON serializer
func CreateMessage(out MessageOut) Message {
	b, err := json.Marshal(out.Message)
	if err != nil {
		panic(err)
	}

	return Message{
		Timestamp:   out.Timestamp,
		From:        out.From,
		To:          out.To,
		ID:          out.ID,
		MessageType: out.MessageType,
		Message:     string(b),
	}
}
