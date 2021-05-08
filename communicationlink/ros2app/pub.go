package ros2app

import (
	"log"

	"github.com/tiiuae/rclgo/pkg/ros2"
	std_msgs "github.com/tiiuae/rclgo/pkg/ros2/msgs/std_msgs/msg"
	"github.com/tiiuae/rclgo/pkg/ros2/ros2_type_dispatcher"
	"github.com/tiiuae/rclgo/pkg/ros2/ros2types"
)

func NewPublisher(rclNode *ros2.Node, topicName string, messageType string) *ros2.Publisher {
	ros2msg, ok := ros2_type_dispatcher.TranslateROS2MsgTypeNameToType(messageType)
	if !ok {
		log.Fatalf("Unable to map message type: %s", messageType)
	}
	pub, err := rclNode.NewPublisher(topicName, ros2msg)
	if err != nil {
		log.Fatalf("Unable to create publisher: %v", err)
	}

	return pub
}

func CreateString(value string) ros2types.ROS2Msg {
	rosmsg := std_msgs.NewString()
	rosmsg.Data.SetDefaults(value)
	return rosmsg
}
