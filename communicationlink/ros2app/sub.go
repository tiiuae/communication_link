package ros2app

import (
	"context"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/tiiuae/rclgo/pkg/ros2"
	"github.com/tiiuae/rclgo/pkg/ros2/ros2_type_dispatcher"
)

type Subscription struct {
	TopicName   string
	MessageType string
	Handler     ros2.SubscriptionCallback
}

type Subscriptions struct {
	rclNode       *ros2.Node
	subscriptions []*Subscription
}

func (ss *Subscriptions) Add(topicName string, messageType string, subscriptionCallback ros2.SubscriptionCallback) {
	ss.subscriptions = append(ss.subscriptions, &Subscription{topicName, messageType, subscriptionCallback})
}

func NewSubscriptions(rclNode *ros2.Node) *Subscriptions {
	return &Subscriptions{rclNode, make([]*Subscription, 0)}
}

func (ss *Subscriptions) Subscribe(ctx context.Context) error {
	for _, s := range ss.subscriptions {
		ros2msg, ok := ros2_type_dispatcher.TranslateROS2MsgTypeNameToType(s.MessageType)
		if !ok {
			return errors.Errorf("Unable to map message type: %s", s.MessageType)
		}
		sub, err := ss.rclNode.NewSubscription(s.TopicName, ros2msg.Clone(), s.Handler)
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
