package flyf4f

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

type f4ftask struct {
	ID            string
	InstanceCount int
	SeqReached    int
	Valid         bool
	Finished      bool
	State         taskState
	Points        []types.Point
}

type armState int

const (
	armInitializing armState = iota
	armStandby
	armArming
	armArmed
	armDisarming
)

type flightState int

const (
	flightGround flightState = iota
	flightTakingOff
	flightAirborne
	flightLanding
)

type state struct {
	armState             armState
	flightState          flightState
	lastVehicleState     types.VehicleState
	lastNavigationStatus NavigationStatus
	tasks                []*f4ftask
}

func newState() *state {
	return &state{
		armInitializing,
		flightGround,
		types.VehicleState{ArmingState: types.ArmingStateInit0, NavigationState: types.NavigationStateLandMode18},
		NavigationStatus{Status: "IDLE"},
		make([]*f4ftask, 0),
	}
}

func (s *state) handleVehicleState(msg types.VehicleState) {
	s.lastVehicleState = msg
	switch msg.ArmingState {
	case types.ArmingStateStandBy1:
		s.armState = armStandby
	case types.ArmingStateArmed2:
		s.armState = armArmed
	}
	switch msg.NavigationState {
	case types.NavigationStateAutoMissionMode3:
		s.flightState = flightAirborne
	// case types.NavigationStateAutoLoiterMode4:
	// 	s.flightState = flightAirborne
	case types.NavigationStateTakeoffMode17:
		s.flightState = flightTakingOff
	case types.NavigationStateLandMode18:
		s.flightState = flightGround
	}
}

func (s *state) handleNavigationStatus(msg NavigationStatus) {
	lastStatus := s.lastNavigationStatus.Status
	s.lastNavigationStatus = msg
	task := s.getActiveTask()
	if task == nil {
		return
	}

	if lastStatus != "IDLE" && msg.Status == "IDLE" {
		task.State = taskUpdated
		task.Finished = true
	}
}

func (s *state) handleFlyPath(msg types.FlyPath) {
	points := make([]types.Point, 0)
	for _, x := range msg.Points {
		points = append(points, types.Point{X: x.X, Y: x.Y, Z: x.Z})
	}
	task := f4ftask{
		msg.ID,
		-1,
		-1,
		true,
		false,
		taskCreated,
		points,
	}
	s.tasks = append(s.tasks, &task)
}

func (s *state) process() []types.Message {
	task := s.getActiveTask()
	if s.armState == armInitializing || s.armState == armArming || s.armState == armDisarming {
		if task != nil {
			log.Printf("F4F: task queued, vehicle initializing")
		}
		return []types.Message{}
	}

	if task == nil {
		log.Printf("F4F: no active task")
		return []types.Message{}
	}

	result := make([]types.Message, 0)
	switch task.State {
	case taskCreated:
		if s.armState == armStandby {
			result = append(result, createLocalMessage(Arm{}))
			s.armState = armArming
		} else if s.armState == armArmed && s.flightState == flightGround {
			result = append(result, createLocalMessage(TakeOff{}))
			s.flightState = flightTakingOff
		} else if s.armState == armArmed && s.flightState == flightAirborne {
			result = append(result, createLocalMessage(types.SendPath{Points: task.Points}))
			task.State = taskSent
		}
	// case taskAccepted:
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

func (s *state) getActiveTask() *f4ftask {
	for _, t := range s.tasks {
		if t.State == taskCompleted || t.State == taskFailed {
			continue
		}
		return t
	}

	return nil
}

func createLocalMessage(msg interface{}) types.Message {
	return types.CreateMessage("", "", "", msg)
}
