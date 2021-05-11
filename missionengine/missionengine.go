package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tiiuae/communication_link/missionengine/gittransport"
	"github.com/tiiuae/communication_link/missionengine/worldengine"
	"github.com/tiiuae/rclgo/pkg/ros2"
	nav_msgs "github.com/tiiuae/rclgo/pkg/ros2/msgs/nav_msgs/msg"
	px4_msgs "github.com/tiiuae/rclgo/pkg/ros2/msgs/px4_msgs/msg"
	std_msgs "github.com/tiiuae/rclgo/pkg/ros2/msgs/std_msgs/msg"

	msg "github.com/tiiuae/communication_link/missionengine/types"
	"github.com/tiiuae/rclgo/pkg/ros2/ros2types"
)

const (
	TOPIC_MISSION_ENGINE = "missionengine"
	TOPIC_MISSION_RESULT = "MissionResult_PubSubTopic"
)

type MissionEngine struct {
	ctx           context.Context
	wg            *sync.WaitGroup
	localNode     *ros2.Node
	fleetNode     *ros2.Node
	droneName     string
	updateBacklog chan struct{}
}

func New(ctx context.Context, wg *sync.WaitGroup, localNode *ros2.Node, fleetNode *ros2.Node, droneName string) *MissionEngine {
	return &MissionEngine{ctx, wg, localNode, fleetNode, droneName, make(chan struct{})}
}

func (me *MissionEngine) Start(gitServerAddress string, gitServerKey string) {
	ctx := me.ctx
	wg := me.wg
	droneName := me.droneName

	go func() {
		defer func() {
			r := recover()
			if r != nil {
				log.Printf("Recover: %v", r)
			}
		}()
		log.Printf("Starting mission engine of drone: '%s'", droneName)
		log.Printf("Running git clone...")
		gt := gittransport.New(gitServerAddress, gitServerKey, droneName)

		we := worldengine.New(droneName)

		messages := make(chan msg.Message, 10)
		go runMessageLoop(ctx, wg, we, me.localNode, me.fleetNode, messages)
		go runGitTransport(ctx, wg, gt, messages, me.updateBacklog, droneName)
		go runMissionEngineSubscriber(ctx, wg, messages, me.fleetNode, droneName)
		go runMissionResultSubscriber(ctx, wg, messages, me.localNode, droneName)
	}()
}

func (me *MissionEngine) UpdateBacklog() {
	me.updateBacklog <- struct{}{}
}

func runMessageLoop(ctx context.Context, wg *sync.WaitGroup, we *worldengine.WorldEngine, localNode *ros2.Node, fleetNode *ros2.Node, ch <-chan msg.Message) {
	wg.Add(1)
	defer wg.Done()

	pub, err := fleetNode.NewPublisher(TOPIC_MISSION_ENGINE, &std_msgs.String{})
	if err != nil {
		log.Fatalf("Failed to create publisher: %v", err)
	}
	pubpath, err := localNode.NewPublisher("path", &nav_msgs.Path{})
	if err != nil {
		log.Fatalf("Failed to create publisher: %v", err)
	}
	pubmavlink, err := localNode.NewPublisher("mavlinkcmd", &std_msgs.String{})
	if err != nil {
		log.Fatalf("Failed to create publisher: %v", err)
	}

	defer pub.Fini()
	defer pubpath.Fini()
	defer pubmavlink.Fini()

	for {
		select {
		case <-ctx.Done():
			return
		case msgIn := <-ch:
			log.Printf("Message received: %s: %s", msgIn.MessageType, msgIn.Message)
			messagesOut := we.HandleMessage(msgIn, pubpath, pubmavlink)
			for _, msgOut := range messagesOut {
				switch m := msgOut.Message.(type) {
				case worldengine.FlightPath:
					path := createPath(m)
					pubpath.Publish(path)
				case worldengine.StartMission:
					go func() {
						time.Sleep(m.Delay)
						pubmavlink.Publish(createString("start_mission"))
					}()
				case []*worldengine.FlightPlan:
					publishFleetMessage(pub, msgOut)
				case []*worldengine.MissionPlan:
					publishFleetMessage(pub, msgOut)
				case worldengine.TasksAssigned:
					publishFleetMessage(pub, msgOut)
				case worldengine.TaskCompleted:
					publishFleetMessage(pub, msgOut)
				default:
					log.Fatalf("Unkown message type: %T", msgOut.Message)
				}
			}
		}
	}
}

func publishFleetMessage(pub *ros2.Publisher, msgOut msg.MessageOut) {
	m := msg.CreateMessage(msgOut)
	log.Printf("Message out: %v", m)
	b, err := json.Marshal(m)
	if err != nil {
		panic("Unable to marshal missionengine message")
	}
	pub.Publish(createString(string(b)))
}

func runGitTransport(ctx context.Context, wg *sync.WaitGroup, gt *gittransport.GitEngine, ch chan<- msg.Message, gitPull <-chan struct{}, droneName string) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			close(ch)
			return
		case <-gitPull:
		case <-time.After(5 * time.Second):
		}
		msgs := gt.PullMessages(droneName)
		for _, msg := range msgs {
			ch <- msg
		}
	}
}

func runMissionEngineSubscriber(ctx context.Context, wg *sync.WaitGroup, ch chan<- msg.Message, node *ros2.Node, droneName string) {
	wg.Add(1)
	defer wg.Done()

	sub, rclErr := node.NewSubscription(TOPIC_MISSION_ENGINE, &std_msgs.String{}, func(s *ros2.Subscription) {
		var m std_msgs.String
		_, rlcErr := s.TakeMessage(&m)
		if rlcErr != nil {
			log.Print("TakeMessage failed: runMissionEngineSubscriber")
			return
		}

		str := fmt.Sprintf("%v", m.Data)
		var msg msg.Message
		err := json.Unmarshal([]byte(str), &msg)
		if err != nil {
			panic("Unable to unmarshal missionengine message")
		}
		ch <- msg
	})

	if rclErr != nil {
		log.Fatalf("Unable to subscribe to topic 'missions': %v", rclErr)
	}

	err := sub.Spin(ctx, 5*time.Second)
	if err != nil {
		log.Printf("Subscription failed: %v", err)
	}
}

func runMissionResultSubscriber(ctx context.Context, wg *sync.WaitGroup, ch chan<- msg.Message, node *ros2.Node, droneName string) {
	wg.Add(1)
	defer wg.Done()

	sub, rclErr := node.NewSubscription(TOPIC_MISSION_RESULT, &px4_msgs.MissionResult{}, func(s *ros2.Subscription) {
		var m px4_msgs.MissionResult
		_, rlcErr := s.TakeMessage(&m)
		if rlcErr != nil {
			log.Print("TakeMessage failed: runMissionEngineSubscriber")
			return
		}

		b, err := json.Marshal(m)
		if err != nil {
			panic("Unable to marshal MissionResult")
		}
		ch <- msg.Message{
			Timestamp:   time.Now().UTC(),
			From:        droneName,
			To:          droneName,
			ID:          "",
			MessageType: "mission-result",
			Message:     string(b),
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

func createString(value string) ros2types.ROS2Msg {
	rosmsg := std_msgs.NewString()
	rosmsg.Data.SetDefaults(value)
	return rosmsg
}
