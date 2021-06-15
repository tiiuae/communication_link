package flypx4

import (
	"log"

	"github.com/tiiuae/communication_link/missionengine/internal/types"
)

type taskState int

const (
	taskCreated taskState = iota
	taskSent
	taskAccepted
	taskRejected
	taskStarted
	taskUpdated
	taskRunning
	taskFailed
	taskCompleted
)

type px4task struct {
	ID            string
	InstanceCount int
	SeqReached    int
	Valid         bool
	Finished      bool
	State         taskState
	Points        []types.Point
}

type state struct {
	lastVehicleState  types.VehicleState
	lastMissionResult types.MissionResult
	tasks             []*px4task
}

func newState() *state {
	return &state{
		types.VehicleState{ArmingState: types.ArmingStateInit0, NavigationState: types.NavigationStateLandMode18},
		types.MissionResult{InstanceCount: -1},
		make([]*px4task, 0),
	}
}

func (s *state) handleVehicleState(msg types.VehicleState) {
	s.lastVehicleState = msg
}

func (s *state) handleMissionResult(msg types.MissionResult) {
	s.lastMissionResult = msg

	// InstanceCount
	// 	- usually starts from 1 when no path is yet sent
	//  - increments by 1 every time a new path is sent
	// 	- sometimes increments randomly by itself
	//	  -> doesn't match to any path sent, but SeqReached seems to be -1 always
	task := s.getTaskByInstanceCount(msg.InstanceCount)
	if task == nil {
		log.Printf("MissionResult: no matching task found for instance count: %d", msg.InstanceCount)
		return
	}
	if task.InstanceCount == -1 {
		log.Printf("MissionResult: matching instance count: %d to task %s", msg.InstanceCount, task.ID)
		task.InstanceCount = msg.InstanceCount
	}
	task.SeqReached = msg.SeqReached
	task.Valid = msg.Valid
	task.Finished = msg.Finished
	if task.State == taskSent {
		if msg.Valid == false {
			task.State = taskRejected
			return
		}
		task.State = taskAccepted
	}

	// SeqReached: -1, 0, ... N
	// Continue if SeqReached is: 2 (point-1 reached), 5 (point-2 reached)...
	if msg.SeqReached%3 != 2 {
		return
	}

	// TODO: progress notifications

	task.State = taskUpdated
}

func (s *state) handleFlyPath(msg types.FlyPath) {
	p := px4task{
		msg.ID,
		-1,
		-1,
		true,
		false,
		taskCreated,
		msg.Points,
	}
	s.tasks = append(s.tasks, &p)
}

func (s *state) process() []types.Message {
	task := s.getActiveTask()
	if s.lastVehicleState.ArmingState == types.ArmingStateInit0 {
		if task != nil {
			log.Printf("PX4 task queued, vehicle initializing")
		}
		return []types.Message{}
	}

	if task == nil {
		log.Printf("PX4: no active task")
		return []types.Message{}
	}

	result := make([]types.Message, 0)
	switch task.State {
	case taskCreated:
		result = append(result, types.CreateMessage("send-path", "self", "self", types.SendPath{Points: task.Points}))
		task.State = taskSent
	case taskAccepted:
		result = append(result, types.CreateMessage("start-mission", "self", "self", types.StartMission{}))
		task.State = taskStarted
	case taskRejected:
		out := types.FlyPathProgress{
			ID:            task.ID,
			Index:         -1,
			PathCompleted: false,
			Failed:        true,
		}
		result = append(result, types.CreateMessage("fly-path-progress", "self", "self", out))
		task.State = taskFailed
	case taskUpdated:
		if task.Finished {
			out := types.FlyPathProgress{
				ID:            task.ID,
				Index:         -1,
				PathCompleted: true,
				Failed:        false,
			}
			result = append(result, types.CreateMessage("fly-path-progress", "self", "self", out))
			task.State = taskCompleted
		} else {
			task.State = taskRunning
		}
	}

	return result
}

func (s *state) getTaskByInstanceCount(instanceCount int) *px4task {
	if len(s.tasks) == 0 {
		return nil
	}

	for _, t := range s.tasks {
		if t.InstanceCount == instanceCount {
			// Match found
			return t
		}
		if t.State > taskCreated && t.State < taskUpdated {
			// First unmatched task
			return t
		}
	}

	return nil
}

func (s *state) getActiveTask() *px4task {
	for _, t := range s.tasks {
		if t.State == taskCompleted || t.State == taskFailed {
			continue
		}
		return t
	}

	return nil
}
