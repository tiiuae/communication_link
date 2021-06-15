package gittransport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	types "github.com/tiiuae/communication_link/missionengine/internal/types"
)

type GitTransport struct {
	deviceID         string
	inbox            chan types.Message
	gitServerAddress string
	gitServerKey     string
	dataDir          string
	fileChanges      map[string]time.Time
	filePositions    map[string]int64
}

func New(deviceID string) types.MessageHandler {
	return &GitTransport{
		deviceID,
		make(chan types.Message, 10),
		"",
		"",
		"",
		make(map[string]time.Time),
		make(map[string]int64),
	}
}

func (g *GitTransport) Run(ctx context.Context, wg *sync.WaitGroup, post types.PostFn) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			log.Println("GitTransport shutting down")
			return
		case <-time.After(15 * time.Second):
			g.pullAndSend(post)
		case msg := <-g.inbox:
			switch m := msg.Message.(type) {
			case types.JoinMission:
				err := g.joinMission(m)
				if err != nil {
					log.Println(err)
				}
			case types.UpdateBacklog:
				if g.gitServerAddress == "" {
					break
				}
				g.pullAndSend(post)
			case types.LeaveMission:
				g.leaveMission()
			}
		}
	}
}

func (g *GitTransport) Receive(message types.Message) {
	g.inbox <- message
}

func (g *GitTransport) joinMission(join types.JoinMission) error {
	g.gitServerAddress = join.GitServer
	g.gitServerKey = join.GitServerKey
	g.dataDir = fmt.Sprintf("db/%s-%s", time.Now().Format("20060102150405"), g.deviceID)

	return g.cloneRepository()
}

func (g *GitTransport) leaveMission() {
	g.gitServerAddress = ""
	g.gitServerKey = ""
	g.dataDir = ""
}

func (g *GitTransport) pullAndSend(post types.PostFn) {
	if g.gitServerAddress == "" {
		return
	}

	messages := g.pullMessages()
	if len(messages) == 0 {
		return
	}

	for _, msg := range messages {
		switch msg.MessageType {
		case "drone-added":
			var m types.DroneAdded
			json.Unmarshal([]byte(msg.Message.(string)), &m)
			post(msg.Replace(m))
		case "drone-removed":
			var m types.DroneRemoved
			json.Unmarshal([]byte(msg.Message.(string)), &m)
			post(msg.Replace(m))
		case "task-created":
			var temp types.TaskCreated
			json.Unmarshal([]byte(msg.Message.(string)), &temp)
			switch temp.Type {
			case "fly-to":
				var inner types.FlyToTaskCreated
				json.Unmarshal([]byte(msg.Message.(string)), &inner)
				post(msg.Replace(inner))
			case "execute-preplanned":
				var inner types.ExecutePredefinedTaskCreated
				json.Unmarshal([]byte(msg.Message.(string)), &inner)
				post(msg.Replace(inner))
			}
		}
	}

	// TODO
	post(types.CreateMessage("process", g.deviceID, g.deviceID, types.Process{}))
}
