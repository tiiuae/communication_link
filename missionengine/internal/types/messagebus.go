package types

import (
	"context"
	"log"
	"sync"
)

type PostFn = func(msg Message)

type MessageHandler interface {
	Run(ctx context.Context, wg *sync.WaitGroup, post PostFn)
	Receive(message Message)
}

type MessageBus struct {
	bus       chan Message
	receivers []MessageHandler
}

func NewMessageBus(bus chan Message, receivers ...MessageHandler) *MessageBus {
	return &MessageBus{bus, receivers}
}

func (mb *MessageBus) Run(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	busCapacity := cap(mb.bus)
	post := func(msg Message) {
		busLen := len(mb.bus)
		if busLen > busCapacity/2 {
			log.Printf("WARNING: Bus capacity over 50%% [ %d / %d ]", busLen, busCapacity)
		}
		mb.bus <- msg
	}

	for _, x := range mb.receivers {
		go x.Run(ctx, wg, post)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-mb.bus:
			for _, x := range mb.receivers {
				x.Receive(msg)
			}
		}
	}
}
