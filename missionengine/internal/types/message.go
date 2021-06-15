package types

import (
	"encoding/json"
	"time"
)

type Message struct {
	Timestamp   time.Time   `json:"timestamp"`
	From        string      `json:"from"`
	To          string      `json:"to"`
	ID          string      `json:"id"`
	MessageType string      `json:"message_type"`
	Message     interface{} `json:"message"`
}

type StringMessage struct {
	Timestamp   time.Time `json:"timestamp"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	ID          string    `json:"id"`
	MessageType string    `json:"message_type"`
	Message     string    `json:"message"`
}

// Serialize message to json-message for git and ros transport
func (message *Message) ToJsonMessage() Message {
	b, err := json.Marshal(message.Message)
	if err != nil {
		panic(err)
	}

	return Message{
		Timestamp:   message.Timestamp,
		From:        message.From,
		To:          message.To,
		ID:          message.ID,
		MessageType: message.MessageType,
		Message:     string(b),
	}
}

func (message *StringMessage) Replace(v interface{}) Message {
	return Message{
		message.Timestamp,
		message.From,
		message.To,
		message.ID,
		message.MessageType,
		v,
	}
}

func (message *Message) Replace(v interface{}) Message {
	return Message{
		message.Timestamp,
		message.From,
		message.To,
		message.ID,
		message.MessageType,
		v,
	}
}

func CreateMessage(messageType, from, to string, message interface{}) Message {
	return Message{
		time.Now(),
		from,
		to,
		"id-1",
		messageType,
		message,
	}
}
