package worldengine

import (
	"encoding/json"
	"log"
	"time"

	"github.com/tiiuae/communication_link/missionengine/types"
	"github.com/tiiuae/rclgo/pkg/ros2"
)

type WorldEngine struct {
	me             string
	currentMission string
	state          *worldState
}

func New(me string) *WorldEngine {
	return &WorldEngine{me, "", nil}
}

func (we *WorldEngine) HandleMessage(msg types.Message, pubPath *ros2.Publisher, pubMavlink *ros2.Publisher) []types.MessageOut {
	state := we.state
	outgoing := make([]types.MessageOut, 0)
	if msg.MessageType == "join-mission" {
		we.state = createState(we.me)
		we.currentMission = msg.Message
		return []types.MessageOut{createJoinedMission(we.me, we.currentMission)}
	}

	if we.state == nil {
		log.Printf("Message skipped '%s': no active mission", msg.MessageType)
		return outgoing
	}

	if msg.MessageType == "leave-mission" {
		res := []types.MessageOut{createLeftMission(we.me, we.currentMission)}
		we.state = nil
		we.currentMission = ""
		return res
	} else if msg.MessageType == "drone-added" {
		var message DroneAdded
		json.Unmarshal([]byte(msg.Message), &message)
		outgoing = state.handleDroneAdded(message)
	} else if msg.MessageType == "drone-removed" {
		var message DroneRemoved
		json.Unmarshal([]byte(msg.Message), &message)
		outgoing = state.handleDroneRemoved(message)
	} else if msg.MessageType == "task-created" {
		var message TaskCreated
		json.Unmarshal([]byte(msg.Message), &message)
		if message.Type == "fly-to" {
			var inner FlyToTaskCreated
			json.Unmarshal([]byte(msg.Message), &inner)
			outgoing = state.handleFlyToTaskCreated(inner)
		} else if message.Type == "execute-preplanned" {
			var inner ExecutePredefinedTaskCreated
			json.Unmarshal([]byte(msg.Message), &inner)
			outgoing = state.handleExecutePredefinedToTaskCreated(inner)
		}
	} else if msg.MessageType == "tasks-assigned" {
		var message TasksAssigned
		json.Unmarshal([]byte(msg.Message), &message)
		outgoing = state.handleTasksAssigned(message)
	} else if msg.MessageType == "task-completed" {
		var message TaskCompleted
		json.Unmarshal([]byte(msg.Message), &message)
		outgoing = state.handleTaskCompleted(message)
	} else if msg.MessageType == "mission-result" {
		var message MissionResult
		json.Unmarshal([]byte(msg.Message), &message)
		outgoing = state.handleMissionResult(message)
	} else if msg.MessageType == "process" {
		outgoing = state.process()
	} else {
		log.Printf("Message skipped '%s': unknown", msg.MessageType)
		return outgoing
	}

	return outgoing
}

func createJoinedMission(droneName string, missionSlug string) types.MessageOut {
	return types.MessageOut{
		Timestamp:   time.Now(),
		From:        droneName,
		To:          "self",
		ID:          "id1",
		MessageType: "joined-mission",
		Message: JoinedMission{
			MissionSlug: missionSlug,
		},
	}
}

func createLeftMission(droneName string, missionSlug string) types.MessageOut {
	return types.MessageOut{
		Timestamp:   time.Now(),
		From:        droneName,
		To:          "self",
		ID:          "id1",
		MessageType: "left-mission",
		Message: JoinedMission{
			MissionSlug: missionSlug,
		},
	}
}

// Message types

type DroneAdded struct {
	Name string `json:"name"`
}

type DroneRemoved struct {
	Name string `json:"name"`
}

type TaskCreated struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type FlyToTaskCreated struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Payload struct {
		X float64 `json:"lat"`
		Y float64 `json:"lon"`
		Z float64 `json:"alt"`
	} `json:"payload"`
}

type ExecutePredefinedTaskCreated struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Payload struct {
		Drone string `json:"drone"`
	} `json:"payload"`
}

type TasksAssigned struct {
	Tasks map[string][]*TaskAssignment `json:"tasks"`
}

type TaskAssignment struct {
	ID   string  `json:"id"`
	Type string  `json:"type"`
	X    float64 `json:"lat"`
	Y    float64 `json:"lon"`
	Z    float64 `json:"alt"`
}

type MissionPlan struct {
	ID         string `json:"id"`
	AssignedTo string `json:"assigned_to"`
	Status     string `json:"status"`
}

type FlightPlan struct {
	Reached bool    `json:"reached"`
	X       float64 `json:"lat"`
	Y       float64 `json:"lon"`
	Z       float64 `json:"alt"`
}

type TaskCompleted struct {
	ID string `json:"id"`
}

type MissionResult struct {
	Timestamp           uint64 // time since system start (microseconds)
	InstanceCount       int    // Instance count of this mission. Increments monotonically whenever the mission is modified
	SeqReached          int    // Sequence of the mission item which has been reached, default -1
	SeqCurrent          int    // Sequence of the current mission item
	SeqTotal            int    // Total number of mission items
	Valid               bool   // true if mission is valid
	Warning             bool   // true if mission is valid, but has potentially problematic items leading to safety warnings
	Finished            bool   // true if mission has been completed
	Failure             bool   // true if the mission cannot continue or be completed for some reason
	StayInFailsafe      bool   // true if the commander should not switch out of the failsafe mode
	FlightTermination   bool   // true if the navigator demands a flight termination from the commander app
	ItemDoJumpChanged   bool   // true if the number of do jumps remaining has changed
	ItemChangedIndex    uint16 // indicate which item has changed
	ItemDoJumpRemaining uint16 // set to the number of do jumps remaining for that item
	ExecutionMode       uint8  // indicates the mode in which the mission is executed
}

type FlightPath struct {
	Points []Point `json:"points"`
}

type StartMission struct {
	Delay time.Duration `json:"delay"`
}

type Land struct {
}

type JoinedMission struct {
	MissionSlug string `json:"mission_slug"`
}

type LeftMission struct {
	MissionSlug string `json:"mission_slug"`
}
