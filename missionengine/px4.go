package main

import (
	"github.com/tiiuae/communication_link/missionengine/worldengine"
	builtin_interfaces "github.com/tiiuae/rclgo/pkg/ros2/msgs/builtin_interfaces/msg"
	geometry_msgs "github.com/tiiuae/rclgo/pkg/ros2/msgs/geometry_msgs/msg"
	nav_msgs "github.com/tiiuae/rclgo/pkg/ros2/msgs/nav_msgs/msg"
	std_msgs "github.com/tiiuae/rclgo/pkg/ros2/msgs/std_msgs/msg"
	"github.com/tiiuae/rclgo/pkg/ros2/ros2types"
)

func createPath(flightPath worldengine.FlightPath) *nav_msgs.Path {
	path := nav_msgs.NewPath()
	path.Header = *std_msgs.NewHeader()
	path.Header.Stamp = *builtin_interfaces.NewTime()
	path.Header.Stamp.Sec = 10000
	path.Header.Stamp.Nanosec = 100000
	path.Header.FrameId = "map"
	path.Poses = make([]geometry_msgs.PoseStamped, len(flightPath.Points))
	for i, p := range flightPath.Points {
		point := geometry_msgs.NewPoint()
		point.X = p.X
		point.Y = p.Y
		point.Z = p.Z
		pose := geometry_msgs.NewPoseStamped()
		pose.Header = *std_msgs.NewHeader()
		pose.Header.Stamp = *builtin_interfaces.NewTime()
		pose.Header.Stamp.Sec = 10000
		pose.Header.Stamp.Nanosec = 100000
		pose.Header.FrameId = "map"
		pose.Pose.Position = *point
		path.Poses[i] = *pose
	}

	return path
}

func createString(value string) ros2types.ROS2Msg {
	rosmsg := std_msgs.NewString()
	rosmsg.Data.SetDefaults(value)
	return rosmsg
}
