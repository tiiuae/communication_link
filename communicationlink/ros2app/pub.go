package ros2app

import (
	"log"

	std_msgs "github.com/tiiuae/rclgo-msgs/std_msgs/msg"
	"github.com/tiiuae/rclgo/pkg/rclgo"
	"github.com/tiiuae/rclgo/pkg/rclgo/typemap"
	"github.com/tiiuae/rclgo/pkg/rclgo/types"
)

func NewPublisher(rclNode *rclgo.Node, topicName string, messageType string) *rclgo.Publisher {
	ros2msg, ok := typemap.GetMessage(messageType)
	if !ok {
		log.Fatalf("Unable to map message type: %s", messageType)
	}
	opts := rclgo.NewDefaultPublisherOptions()
	opts.Qos.Reliability = rclgo.RmwQosReliabilityPolicySystemDefault
	pub, err := rclNode.NewPublisher(topicName, ros2msg, opts)
	if err != nil {
		log.Fatalf("Unable to create publisher: %v", err)
	}

	return pub
}

func CreateString(value string) types.Message {
	rosmsg := std_msgs.NewString()
	rosmsg.Data = value
	return rosmsg
}
