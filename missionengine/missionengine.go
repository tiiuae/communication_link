package main

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/tiiuae/communication_link/missionengine/gittransport"
	msg "github.com/tiiuae/communication_link/missionengine/types"
	"github.com/tiiuae/communication_link/missionengine/worldengine"
	"github.com/tiiuae/communication_link/ros"
	types "github.com/tiiuae/communication_link/types"
)

const (
	TOPIC_MISSION_ENGINE = "missionengine"
	TOPIC_MISSION_RESULT = "MissionResult_PubSubTopic"
)

type MissionEngine struct {
	ctx           context.Context
	wg            *sync.WaitGroup
	localNode     *ros.Node
	fleetNode     *ros.Node
	droneName     string
	updateBacklog chan struct{}
}

func New(ctx context.Context, wg *sync.WaitGroup, localNode *ros.Node, fleetNode *ros.Node, droneName string) *MissionEngine {
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

func runMessageLoop(ctx context.Context, wg *sync.WaitGroup, we *worldengine.WorldEngine, localNode *ros.Node, fleetNode *ros.Node, ch <-chan msg.Message) {
	wg.Add(1)
	defer wg.Done()

	pub := fleetNode.InitPublisher(TOPIC_MISSION_ENGINE, "std_msgs/msg/String", (*types.String)(nil))
	pubpath := localNode.InitPublisher("path", "nav_msgs/msg/Path", (*types.Path)(nil))
	pubmavlink := localNode.InitPublisher("mavlinkcmd", "std_msgs/msg/String", (*types.String)(nil))
	defer pub.Finish()
	defer pubpath.Finish()
	defer pubmavlink.Finish()

	for {
		select {
		case <-ctx.Done():
			return
		case m := <-ch:
			log.Printf("Message received: %s: %s", m.MessageType, m.Message)
			messagesOut := we.HandleMessage(m, pubpath, pubmavlink)
			for _, r := range messagesOut {
				log.Printf("Message out: %v", r)
				b, err := json.Marshal(r)
				if err != nil {
					panic("Unable to marshal missionengine message")
				}
				pub.DoPublish(types.GenerateString(string(b)))
				time.Sleep(200 * time.Millisecond) // TODO: remove when rclgo in use
			}
		}
	}
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

func runMissionEngineSubscriber(ctx context.Context, wg *sync.WaitGroup, ch chan<- msg.Message, node *ros.Node, droneName string) {
	wg.Add(1)
	defer wg.Done()
	rosMessages := make(chan types.String)
	sub := node.InitSubscriber(rosMessages, TOPIC_MISSION_ENGINE, "std_msgs/msg/String")
	go sub.DoSubscribe(ctx)

	for {
		select {
		case <-ctx.Done():
			sub.Finish()
			return
		case rosMsg := <-rosMessages:
			str := rosMsg.GetString()
			var m msg.Message
			err := json.Unmarshal([]byte(str), &m)
			if err != nil {
				panic("Unable to unmarshal missionengine message")
			}
			ch <- m
		}
	}
}

func runMissionResultSubscriber(ctx context.Context, wg *sync.WaitGroup, ch chan<- msg.Message, node *ros.Node, droneName string) {
	wg.Add(1)
	defer wg.Done()
	rosMessages := make(chan types.MissionResult)
	sub := node.InitSubscriber(rosMessages, TOPIC_MISSION_RESULT, "px4_msgs/msg/MissionResult")
	go sub.DoSubscribe(ctx)

	for {
		select {
		case <-ctx.Done():
			sub.Finish()
			return
		case rosMsg := <-rosMessages:
			b, err := json.Marshal(rosMsg)
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
		}
	}
}
