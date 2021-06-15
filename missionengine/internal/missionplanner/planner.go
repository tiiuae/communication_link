package missionplanner

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/tiiuae/communication_link/missionengine/internal/types"
)

type missionPlanner struct {
	me             string
	inbox          chan types.Message
	currentMission string
	state          *state
}

func New(deviceID string) types.MessageHandler {
	return &missionPlanner{deviceID, make(chan types.Message, 10), "", nil}
}

func (mp *missionPlanner) Run(ctx context.Context, wg *sync.WaitGroup, post types.PostFn) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			log.Println("MissionPlanner shutting down")
			return
		case msg := <-mp.inbox:
			out := mp.handleMessage(msg)
			for _, x := range out {
				post(x)
			}
		}
	}
}

func (mp *missionPlanner) Receive(message types.Message) {
	mp.inbox <- message
}

func (mp *missionPlanner) handleMessage(msg types.Message) []types.Message {
	outgoing := make([]types.Message, 0)
	switch m := msg.Message.(type) {
	case types.JoinMission:
		mp.state = createState(mp.me)
		mp.currentMission = m.MissionSlug
		return []types.Message{createJoinedMission(mp.me, mp.currentMission)}
	}

	if mp.state == nil {
		return outgoing
	}

	switch m := msg.Message.(type) {
	case types.LeaveMission:
		res := []types.Message{createLeftMission(mp.me, mp.currentMission)}
		mp.state = nil
		mp.currentMission = ""
		return res
	case types.DroneAdded:
		outgoing = mp.state.handleDroneAdded(m)
	case types.DroneRemoved:
		outgoing = mp.state.handleDroneRemoved(m)
	case types.FlyToTaskCreated:
		outgoing = mp.state.handleFlyToTaskCreated(m)
	case types.ExecutePredefinedTaskCreated:
		outgoing = mp.state.handleExecutePredefinedToTaskCreated(m)
	case types.TasksAssigned:
		outgoing = mp.state.handleTasksAssigned(m)
		outgoing = append(outgoing, mp.state.processTasks()...)
	case types.TaskQueued:
		outgoing = mp.state.handleTaskQueued(m)
	case types.TaskStarted:
		outgoing = mp.state.handleTaskStarted(m)
	case types.TaskCompleted:
		outgoing = mp.state.handleTaskCompleted(m)
		outgoing = append(outgoing, mp.state.processTasks()...)
	case types.TaskFailed:
		outgoing = mp.state.handleTaskFailed(m)
		outgoing = append(outgoing, mp.state.processTasks()...)
	case types.FlyPathProgress:
		outgoing = mp.state.handleFlyPathProgress(m)
	case types.Process:
		outgoing = mp.state.process()
	}

	return outgoing
}

func createJoinedMission(droneName string, missionSlug string) types.Message {
	return types.Message{
		Timestamp:   time.Now(),
		From:        droneName,
		To:          "*",
		ID:          "id1",
		MessageType: "joined-mission",
		Message: types.JoinedMission{
			MissionSlug: missionSlug,
		},
	}
}

func createLeftMission(droneName string, missionSlug string) types.Message {
	return types.Message{
		Timestamp:   time.Now(),
		From:        droneName,
		To:          "*",
		ID:          "id1",
		MessageType: "left-mission",
		Message: types.LeftMission{
			MissionSlug: missionSlug,
		},
	}
}
