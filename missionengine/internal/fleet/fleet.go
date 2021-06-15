package fleet

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tiiuae/communication_link/missionengine/internal/types"
	"github.com/tiiuae/rclgo/pkg/ros2"
	std_msgs "github.com/tiiuae/rclgo/pkg/ros2/msgs/std_msgs/msg"
	"github.com/tiiuae/rclgo/pkg/ros2/ros2types"
)

const TOPIC_MISSION_ENGINE = "missionengine"

type fleet struct {
	fleetNode *ros2.Node
	deviceID  string
	inbox     chan types.Message
}

func New(fleetNode *ros2.Node, deviceID string) types.MessageHandler {
	return &fleet{fleetNode, deviceID, make(chan types.Message, 10)}
}

func (f *fleet) Run(ctx context.Context, wg *sync.WaitGroup, post types.PostFn) {
	go f.runPublisher(ctx, wg)
	go f.runSubscriber(ctx, wg, post)
}

func (f *fleet) Receive(message types.Message) {
	f.inbox <- message
}

func (f *fleet) runPublisher(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	pub, err := f.fleetNode.NewPublisher(TOPIC_MISSION_ENGINE, &std_msgs.String{})
	if err != nil {
		log.Fatalf("Failed to create publisher: %v", err)
	}
	defer pub.Close()

	for {
		select {
		case <-ctx.Done():
			log.Println("Fleet shutting down")
			return
		case msg := <-f.inbox:
			if msg.From == f.deviceID {
				f.broadcastMessageToFleet(pub, msg)
			}
		}
	}
}

func (f *fleet) runSubscriber(ctx context.Context, wg *sync.WaitGroup, post types.PostFn) {
	wg.Add(1)
	defer wg.Done()

	sub, rclErr := f.fleetNode.NewSubscription(TOPIC_MISSION_ENGINE, &std_msgs.String{}, func(s *ros2.Subscription) {
		var m std_msgs.String
		_, rlcErr := s.TakeMessage(&m)
		if rlcErr != nil {
			log.Print("TakeMessage failed: runMissionEngineSubscriber")
			return
		}

		str := fmt.Sprintf("%v", m.Data)
		var msg types.StringMessage
		err := json.Unmarshal([]byte(str), &msg)
		if err != nil {
			panic("Unable to unmarshal missionengine message")
		}

		// Skip messages from this device. They are already handled.
		if msg.From == f.deviceID {
			return
		}

		switch msg.MessageType {
		case "tasks-assigned":
			var m types.TasksAssigned
			json.Unmarshal([]byte(msg.Message), &m)
			post(msg.Replace(m))
		case "task-queued":
			var m types.TaskQueued
			json.Unmarshal([]byte(msg.Message), &m)
			post(msg.Replace(m))
		case "task-started":
			var m types.TaskStarted
			json.Unmarshal([]byte(msg.Message), &m)
			post(msg.Replace(m))
		case "task-completed":
			var m types.TaskCompleted
			json.Unmarshal([]byte(msg.Message), &m)
			post(msg.Replace(m))
		case "task-failed":
			var m types.TaskFailed
			json.Unmarshal([]byte(msg.Message), &m)
			post(msg.Replace(m))
		case "joined-mission":
			var m types.JoinedMission
			json.Unmarshal([]byte(msg.Message), &m)
			post(msg.Replace(m))
		case "left-mission":
			var m types.LeftMission
			json.Unmarshal([]byte(msg.Message), &m)
			post(msg.Replace(m))
		}
	})

	if rclErr != nil {
		log.Fatalf("Unable to subscribe to topic 'missions': %v", rclErr)
	}

	err := sub.Spin(ctx, 5*time.Second)
	if err != nil {
		log.Printf("Subscription failed: %v", err)
	}
}

func (f *fleet) broadcastMessageToFleet(pub *ros2.Publisher, msg types.Message) {
	switch msg.Message.(type) {
	case []*types.FlightPlan:
		publishFleetMessage(pub, msg)
	case []*types.MissionPlan:
		publishFleetMessage(pub, msg)
	case types.TasksAssigned:
		publishFleetMessage(pub, msg)
	case types.TaskQueued:
		publishFleetMessage(pub, msg)
	case types.TaskStarted:
		publishFleetMessage(pub, msg)
	case types.TaskCompleted:
		publishFleetMessage(pub, msg)
	case types.TaskFailed:
		publishFleetMessage(pub, msg)
	case types.JoinedMission:
		publishFleetMessage(pub, msg)
	case types.LeftMission:
		publishFleetMessage(pub, msg)
	}
}

func publishFleetMessage(pub *ros2.Publisher, msg types.Message) {
	m := msg.ToJsonMessage()
	b, err := json.Marshal(m)
	if err != nil {
		panic("Unable to marshal missionengine message")
	}
	pub.Publish(createString(string(b)))
}

func createString(value string) ros2types.ROS2Msg {
	rosmsg := std_msgs.NewString()
	rosmsg.Data.SetDefaults(value)
	return rosmsg
}
