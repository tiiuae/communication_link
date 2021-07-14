package ros2app

import (
	"context"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/tiiuae/rclgo/pkg/rclgo"
	"github.com/tiiuae/rclgo/pkg/rclgo/typemap"
)

type Subscription struct {
	TopicName   string
	MessageType string
	Handler     rclgo.SubscriptionCallback
}

type Subscriptions struct {
	rclNode       *rclgo.Node
	subscriptions []*Subscription
}

func (ss *Subscriptions) Add(topicName string, messageType string, subscriptionCallback rclgo.SubscriptionCallback) {
	ss.subscriptions = append(ss.subscriptions, &Subscription{topicName, messageType, subscriptionCallback})
}

func NewSubscriptions(rclNode *rclgo.Node) *Subscriptions {
	return &Subscriptions{rclNode, make([]*Subscription, 0)}
}

func (ss *Subscriptions) Subscribe(ctx context.Context) error {
	for _, s := range ss.subscriptions {
		ros2msg, ok := typemap.GetMessage(s.MessageType)
		if !ok {
			return errors.Errorf("Unable to map message type: %s", s.MessageType)
		}
		sub, err := ss.rclNode.NewSubscription(s.TopicName, ros2msg, s.Handler)
		if err != nil {
			return errors.WithMessagef(err, "Unable to subscribe to topic %s", s.TopicName)
		}

		go func() {
			err := sub.Spin(ctx, 5*time.Second)
			log.Printf("Subscription failed: %v", err)
		}()
	}

	return nil
}
