package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	msg "github.com/tiiuae/communication_link/missionengine/types"
	ros "github.com/tiiuae/communication_link/ros"
	types "github.com/tiiuae/communication_link/types"
)

type JoinMission struct {
	GitServerAddress string `json:"git_server_address"`
	GitServerKey     string `json:"git_server_key"`
	MissionSlug      string `json:"mission_slug"`
}

func startCommandHandlers(ctx context.Context, wg *sync.WaitGroup, me *MissionEngine, node *ros.Node) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		handleCommands(ctx, me, node)
	}()
}

func handleCommands(ctx context.Context, me *MissionEngine, node *ros.Node) {
	messages := make(chan types.String)
	log.Printf("Creating subscriber for %s", "missions")
	sub := node.InitSubscriber(messages, "missions", "std_msgs/msg/String")
	go sub.DoSubscribe(ctx)
	for m := range messages {
		str := m.GetString()
		var msg msg.Message
		err := json.Unmarshal([]byte(str), &msg)
		if err != nil {
			log.Printf("Could not unmarshal payload: %v", err)
			return
		}
		handleCommand(msg, me)
	}
	sub.Finish()
}

func handleCommand(msg msg.Message, me *MissionEngine) {
	switch msg.MessageType {
	case "join-mission":
		var message JoinMission
		json.Unmarshal([]byte(msg.Message), &message)
		sshUrl := fmt.Sprintf("ssh://git@%s/%s.git", message.GitServerAddress, message.MissionSlug)
		me.Start(sshUrl, message.GitServerKey)
	case "leave-mission":
		// me.Stop();
	case "update-backlog":
		me.UpdateBacklog()
	default:
		log.Printf("Unknown command: %s", msg.MessageType)
	}
}
