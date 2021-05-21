package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"
	types "github.com/tiiuae/communication_link/missionengine/types"
	"github.com/tiiuae/rclgo/pkg/ros2"
	std_msgs "github.com/tiiuae/rclgo/pkg/ros2/msgs/std_msgs/msg"
)

type JoinMission struct {
	GitServerAddress string `json:"git_server_address"`
	GitServerKey     string `json:"git_server_key"`
	MissionSlug      string `json:"mission_slug"`
	SSHID            []byte `json:"ssh_id"`
	SSHKnownHosts    []byte `json:"ssh_known_hosts"`
}

func startCommandHandlers(ctx context.Context, wg *sync.WaitGroup, me *MissionEngine, node *ros2.Node) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		handleCommands(ctx, me, node)
	}()
}

func handleCommands(ctx context.Context, me *MissionEngine, node *ros2.Node) {
	sub, rclErr := node.NewSubscription("missions", &std_msgs.String{}, func(s *ros2.Subscription) { handleCommand(s, me) })
	if rclErr != nil {
		log.Fatalf("Unable to subscribe to topic 'missions': %v", rclErr)
	}

	err := sub.Spin(ctx, 5*time.Second)
	if err != nil {
		log.Printf("Subscription failed: %v", err)
	}
}

func handleCommand(s *ros2.Subscription, me *MissionEngine) {
	var m std_msgs.String
	_, rlcErr := s.TakeMessage(&m)
	if rlcErr != nil {
		log.Print("TakeMessage failed: handleCommand")
		return
	}

	str := fmt.Sprintf("%v", m.Data)

	var msg types.Message
	err := json.Unmarshal([]byte(str), &msg)
	if err != nil {
		log.Printf("Could not unmarshal payload: %v", err)
		return
	}

	switch msg.MessageType {
	case "join-mission":
		var message JoinMission
		err := json.Unmarshal([]byte(msg.Message), &message)
		if err != nil {
			log.Printf("Could not unmarshal payload: %v", err)
			return
		}
		err = storeSSHFiles(message)
		if err != nil {
			log.Printf("Could not store SSH files: %v", err)
			return
		}
		sshUrl := fmt.Sprintf("ssh://git@%s/%s.git", message.GitServerAddress, message.MissionSlug)
		me.JoinMission(message.MissionSlug, sshUrl, message.GitServerKey)
	case "leave-mission":
		me.LeaveMission()
	case "update-backlog":
		me.UpdateBacklog()
	default:
		log.Printf("Unknown command: %s", msg.MessageType)
	}
}

func storeSSHFiles(message JoinMission) error {
	wd, _ := os.Getwd()
	idPath := filepath.Join(wd, "/ssh/id_rsa")
	khPath := filepath.Join(wd, "/ssh/known_host_cloud")

	os.Mkdir("ssh", 0755)
	err := ioutil.WriteFile(idPath, message.SSHID, 0600)
	if err != nil {
		return errors.WithMessage(err, "Could not write id_rsa file")
	}
	err = ioutil.WriteFile(khPath, message.SSHKnownHosts, 0644)
	if err != nil {
		return errors.WithMessage(err, "Could not write known_host_cloud file")
	}

	return nil
}
