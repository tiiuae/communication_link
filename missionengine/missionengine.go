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
)

const (
	TOPIC_MISSION_ENGINE = "missionengine"
	TOPIC_MISSION_RESULT = "MissionResult_PubSubTopic"
)

type MissionEngine struct {
	ctx               context.Context
	wg                *sync.WaitGroup
	localNode         *ros2.Node
	fleetNode         *ros2.Node
	droneName         string
	updateBacklog     chan struct{}
	messages          chan msg.Message
	closeGitTransport context.CancelFunc
}

func New(ctx context.Context, wg *sync.WaitGroup, localNode *ros2.Node, fleetNode *ros2.Node, droneName string) *MissionEngine {
	me := &MissionEngine{
		ctx:               ctx,
		wg:                wg,
		localNode:         localNode,
		fleetNode:         fleetNode,
		droneName:         droneName,
		updateBacklog:     make(chan struct{}),
		messages:          make(chan msg.Message, 10),
		closeGitTransport: nil,
	}

	go func() {
		defer func() {
			r := recover()
			if r != nil {
				log.Printf("Recover: %v", r)
			}
		}()
		log.Printf("Starting mission engine of drone: '%s'", droneName)

		we := worldengine.New(droneName)

		go runMessageLoop(ctx, wg, we, me.localNode, me.fleetNode, me.messages)
		go runMissionEngineSubscriber(ctx, wg, me.messages, me.fleetNode, droneName)
		go runMissionResultSubscriber(ctx, wg, me.messages, me.localNode, droneName)
	}()

	return me
}

func (me *MissionEngine) JoinMission(missionSlug string, gitServerAddress string, gitServerKey string) {
	go func() {
		defer func() {
			r := recover()
			if r != nil {
				log.Printf("Recover: %v", r)
			}
		}()
		log.Printf("Joining mission: '%s'", missionSlug)
		log.Printf("Running git clone...")
		gt, err := gittransport.New(gitServerAddress, gitServerKey, me.droneName)
		if err != nil {
			log.Print(err)
			return
		}

		me.messages <- msg.Message{
			Timestamp:   time.Now(),
			From:        "cloud",
			To:          "*",
			ID:          "id1",
			MessageType: "join-mission",
			Message:     missionSlug,
		}

		ctx, cancel := context.WithCancel(me.ctx)
		me.closeGitTransport = cancel
		go runGitTransport(ctx, me.wg, gt, me.messages, me.updateBacklog, me.droneName, missionSlug)
	}()
}

func (me *MissionEngine) UpdateBacklog() {
	me.updateBacklog <- struct{}{}
}

func (me *MissionEngine) LeaveMission() {
	if me.closeGitTransport != nil {
		me.closeGitTransport()
		me.closeGitTransport = nil
	}

	me.messages <- msg.Message{
		Timestamp:   time.Now(),
		From:        "cloud",
		To:          "*",
		ID:          "id1",
		MessageType: "leave-mission",
		Message:     "",
	}
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

	defer pub.Close()
	defer pubpath.Close()
	defer pubmavlink.Close()

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
				case worldengine.Land:
					go func() {
						pubmavlink.Publish(createString("land"))
					}()
				case []*worldengine.FlightPlan:
					publishFleetMessage(pub, msgOut)
				case []*worldengine.MissionPlan:
					publishFleetMessage(pub, msgOut)
				case worldengine.TasksAssigned:
					publishFleetMessage(pub, msgOut)
				case worldengine.TaskQueued:
					publishFleetMessage(pub, msgOut)
				case worldengine.TaskStarted:
					publishFleetMessage(pub, msgOut)
				case worldengine.TaskCompleted:
					publishFleetMessage(pub, msgOut)
				case worldengine.JoinedMission:
					publishFleetMessage(pub, msgOut)
				case worldengine.LeftMission:
					publishFleetMessage(pub, msgOut)
				default:
					log.Fatalf("Unknown message type: %T", msgOut.Message)
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

func runGitTransport(ctx context.Context, wg *sync.WaitGroup, gt *gittransport.GitEngine, ch chan<- msg.Message, gitPull <-chan struct{}, droneName string, missionSlug string) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			// close(ch)
			log.Printf("Leaving mission: '%s'", missionSlug)
			return
		case <-gitPull:
		case <-time.After(5 * time.Second):
		}
		msgs := gt.PullMessages(droneName)
		for _, msg := range msgs {
			ch <- msg
		}

		// Trigger process when new messages arrive / after latest message is handled
		if len(msgs) > 0 {
			ch <- msg.Message{
				Timestamp:   time.Now().UTC(),
				From:        droneName,
				To:          droneName,
				ID:          "",
				MessageType: "process",
				Message:     "",
			}
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
