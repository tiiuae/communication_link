package flypx4

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tiiuae/communication_link/missionengine/internal/types"
	"github.com/tiiuae/rclgo/pkg/ros2"
	builtin_interfaces "github.com/tiiuae/rclgo/pkg/ros2/msgs/builtin_interfaces/msg"
	geometry_msgs "github.com/tiiuae/rclgo/pkg/ros2/msgs/geometry_msgs/msg"
	nav_msgs "github.com/tiiuae/rclgo/pkg/ros2/msgs/nav_msgs/msg"
	px4_msgs "github.com/tiiuae/rclgo/pkg/ros2/msgs/px4_msgs/msg"
	std_msgs "github.com/tiiuae/rclgo/pkg/ros2/msgs/std_msgs/msg"
	"github.com/tiiuae/rclgo/pkg/ros2/ros2types"
)

type px4 struct {
	localNode *ros2.Node
	deviceID  string
	inbox     chan types.Message
	state     *state
}

func New(localNode *ros2.Node, deviceID string) types.MessageHandler {
	return &px4{localNode, deviceID, make(chan types.Message, 10), newState()}
}

func (px4 *px4) Run(ctx context.Context, wg *sync.WaitGroup, post types.PostFn) {
	go px4.runMessageLoop(ctx, wg, post)
	go px4.runPX4Subscriber(ctx, wg)
}

func (px4 *px4) Receive(message types.Message) {
	px4.inbox <- message
}

func (px4 *px4) runMessageLoop(ctx context.Context, wg *sync.WaitGroup, post types.PostFn) {
	wg.Add(1)
	defer wg.Done()

	pubPath, err := px4.localNode.NewPublisher("path", &nav_msgs.Path{})
	if err != nil {
		log.Fatalf("Failed to create publisher: %v", err)
	}
	pubMavlink, err := px4.localNode.NewPublisher("mavlinkcmd", &std_msgs.String{})
	if err != nil {
		log.Fatalf("Failed to create publisher: %v", err)
	}

	defer pubPath.Close()
	defer pubMavlink.Close()

	for {
		select {
		case <-ctx.Done():
			log.Println("PX4 shutting down")
			return
		case msg := <-px4.inbox:
			switch m := msg.Message.(type) {
			case types.VehicleState:
				px4.state.handleVehicleState(m)
				publishMessages(pubPath, pubMavlink, post, px4.state.process())
			case types.MissionResult:
				px4.state.handleMissionResult(m)
				publishMessages(pubPath, pubMavlink, post, px4.state.process())
			case types.FlyPath:
				px4.state.handleFlyPath(m)
				publishMessages(pubPath, pubMavlink, post, px4.state.process())
			case types.Land:
				publishMessages(pubPath, pubMavlink, post, []types.Message{msg})
			}
		}
	}
}

func publishMessages(pubPath *ros2.Publisher, pubMavlink *ros2.Publisher, post types.PostFn, messages []types.Message) {
	for _, msg := range messages {
		switch m := msg.Message.(type) {
		case types.FlyPathProgress:
			post(msg)
		case types.SendPath:
			path := createPath(m.Points)
			pubPath.Publish(path)
		case types.StartMission:
			pubMavlink.Publish(createString("start_mission"))
		case types.Land:
			pubMavlink.Publish(createString("land"))
		}
	}
}

func (px4 *px4) runPX4Subscriber(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	var missionResultFilter string = ""

	sub, rclErr := px4.localNode.NewSubscription("MissionResult_PubSubTopic", &px4_msgs.MissionResult{}, func(s *ros2.Subscription) {
		var m px4_msgs.MissionResult
		_, rlcErr := s.TakeMessage(&m)
		if rlcErr != nil {
			log.Print("TakeMessage failed: runPX4Subscriber")
			return
		}

		key := fmt.Sprintf("%v-%v-%v-%v", m.InstanceCount, m.SeqReached, m.Valid, m.Finished)
		if key == missionResultFilter {
			return
		}
		missionResultFilter = key

		out := types.Message{
			Timestamp:   time.Now().UTC(),
			From:        px4.deviceID,
			To:          px4.deviceID,
			ID:          "id-1",
			MessageType: "mission-result",
			Message: types.MissionResult{
				Timestamp:           m.Timestamp,
				InstanceCount:       int(m.InstanceCount),
				SeqReached:          int(m.SeqReached),
				SeqCurrent:          int(m.SeqCurrent),
				SeqTotal:            int(m.SeqTotal),
				Valid:               m.Valid,
				Warning:             m.Warning,
				Finished:            m.Finished,
				Failure:             m.Failure,
				StayInFailsafe:      m.StayInFailsafe,
				FlightTermination:   m.FlightTermination,
				ItemDoJumpChanged:   m.ItemDoJumpChanged,
				ItemDoJumpRemaining: m.ItemDoJumpRemaining,
				ExecutionMode:       m.ExecutionMode,
			},
		}

		log.Printf("MissionResult: %+v", out.Message)

		px4.inbox <- out
	})

	if rclErr != nil {
		log.Fatalf("Unable to subscribe to topic 'missions': %v", rclErr)
	}

	err := sub.Spin(ctx, 5*time.Second)
	if err != nil {
		log.Printf("Subscription failed: %v", err)
	}
}

func createString(value string) ros2types.ROS2Msg {
	rosmsg := std_msgs.NewString()
	rosmsg.Data.SetDefaults(value)
	return rosmsg
}

func createPath(points []types.Point) *nav_msgs.Path {
	path := nav_msgs.NewPath()
	path.Header = *std_msgs.NewHeader()
	path.Header.Stamp = *builtin_interfaces.NewTime()
	path.Header.Stamp.Sec = 10000
	path.Header.Stamp.Nanosec = 100000
	path.Header.FrameId = "map"
	path.Poses = make([]geometry_msgs.PoseStamped, len(points))
	for i, p := range points {
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
