package ros2app

import (
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
	rclContext    *ros2.Context
	rclNode       *ros2.Node
	subscriptions []*Subscription
}

func (ss *Subscriptions) Add(topicName string, messageType string, subscriptionCallback ros2.SubscriptionCallback) {
	ss.subscriptions = append(ss.subscriptions, &Subscription{topicName, messageType, subscriptionCallback})
}

func NewSubscriptions(rclContext *ros2.Context, rclNode *ros2.Node) *Subscriptions {
	return &Subscriptions{rclContext, rclNode, make([]*Subscription, 0)}
}

func (ss *Subscriptions) Subscribe() (*ros2.WaitSet, error) {
	subs := make([]*ros2.Subscription, 0)
	for _, s := range ss.subscriptions {
		ros2msg, ok := ros2_type_dispatcher.TranslateROS2MsgTypeNameToType(s.MessageType)
		if !ok {
			return nil, errors.Errorf("Unable to map message type: %s", s.MessageType)
		}
		sub, err := ss.rclNode.NewSubscription(s.TopicName, ros2msg.Clone(), s.Handler)
		if err != nil {
			return nil, errors.WithMessagef(err, "Unable to subscribe to topic %s", s.TopicName)
		}
		subs = append(subs, sub)
	}

	waitSet, err := ss.rclContext.NewWaitSet(subs, nil, 1000*time.Millisecond)
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to create WaitSet")
	}

	return waitSet, nil
}
