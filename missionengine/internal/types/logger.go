package types

import (
	"context"
	"encoding/json"
	"log"
	"sync"
)

type logger struct {
}

func NewLogger() MessageHandler {
	return &logger{}
}

func (l *logger) Receive(message Message) {
	b, _ := json.Marshal(message.Message)

	if message.MessageType == "global-position" {
		return
	}

	log.Printf("Message: %s (%s -> %s): %s", message.MessageType, message.From, message.To, string(b))
}

func (l *logger) Run(ctx context.Context, wg *sync.WaitGroup, post PostFn) {
}
