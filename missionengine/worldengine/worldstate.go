package worldengine

import (
	"encoding/json"
	"log"
	"time"

	"github.com/tiiuae/communication_link/missionengine/ros"
	"github.com/tiiuae/communication_link/missionengine/types"
	rosTypes "github.com/tiiuae/communication_link/missionengine/types"
)

const (
	BACKLOG_ITEM_STATUS_COMPLETED = "completed"
)

type Drones map[string]*droneState
type Backlog []*backlogItem

type worldState struct {
	// My data
	MyName          string
	MyTasks         []*myTask
	MissionInstance int

	// Fleet data
	LeaderName string
	Backlog    Backlog
	Drones     Drones
}

type droneState struct {
	Name  string
	Tasks []*fleetTask
}

type fleetTask struct {
	ID string
}

type myTask struct {
	ID        string
	Completed bool
	Path      []*taskPoint
}

type taskPoint struct {
	Reached bool
	X       float64
	Y       float64
	Z       float64
}

type backlogItem struct {
	ID     string
	Status string
	Type   string
	Drone  string
	X      float64
	Y      float64
	Z      float64
}

func createState(me string) *worldState {
	return &worldState{
		MyName:          me,
		MyTasks:         make([]*myTask, 0),
		MissionInstance: -1,
		LeaderName:      "",
		Backlog:         make([]*backlogItem, 0),
		Drones:          make(map[string]*droneState, 0),
	}
}

func (s *worldState) handleDroneAdded(msg DroneAdded) []types.Message {
	name := msg.Name
	isLeader := len(s.Drones) == 0
	if isLeader {
		s.LeaderName = name
	}
	s.Drones[name] = &droneState{name, []*fleetTask{}}

	// Done if not leader
	if s.LeaderName != s.MyName {
		return []types.Message{}
	}

	if len(s.Backlog) == 0 {
		return []types.Message{}
	}

	// Re-assign tasks
	return s.createTasksAssigned()
}

func (s *worldState) handleDroneRemoved(msg DroneRemoved) []types.Message {
	if msg.Name == s.MyName {
		return []types.Message{}
	}

	delete(s.Drones, msg.Name)

	if msg.Name == s.LeaderName {
		s.LeaderName = smallestString(s.Drones.names())
	}

	// Done if not leader
	if s.LeaderName != s.MyName {
		return []types.Message{}
	}

	if len(s.Backlog) == 0 {
		return []types.Message{}
	}

	// Re-assign tasks
	return s.createTasksAssigned()
}

func (s *worldState) handleFlyToTaskCreated(msg FlyToTaskCreated) []types.Message {
	// Add task to backlog
	bi := backlogItem{
		ID:     msg.ID,
		Status: "",
		Type:   msg.Type,
		Drone:  "",
		X:      msg.Payload.X,
		Y:      msg.Payload.Y,
		Z:      msg.Payload.Z,
	}
	s.Backlog = append(s.Backlog, &bi)

	// Done if not leader
	if s.LeaderName != s.MyName {
		return []types.Message{}
	}

	return s.createTasksAssigned()
}

func (s *worldState) handleExecutePredefinedToTaskCreated(msg ExecutePredefinedTaskCreated) []types.Message {
	// Add task to backlog
	bi := backlogItem{
		ID:     msg.ID,
		Status: "",
		Type:   msg.Type,
		Drone:  msg.Payload.Drone,
		X:      0,
		Y:      0,
		Z:      0,
	}
	s.Backlog = append(s.Backlog, &bi)

	// Done if not leader
	if s.LeaderName != s.MyName {
		return []types.Message{}
	}

	return s.createTasksAssigned()
}

func (s *worldState) handleTasksAssigned(msg TasksAssigned, pubPath *ros.Publisher, pubMavlink *ros.Publisher) []types.Message {
	tasksChanged := false
	for droneName, tasks := range msg.Tasks {
		// Fleet tasks
		drone := s.Drones[droneName]
		drone.Tasks = make([]*fleetTask, 0)
		for _, t := range tasks {
			drone.Tasks = append(drone.Tasks, &fleetTask{ID: t.ID})
		}
		// My tasks
		if droneName == s.MyName {
			newTasks := createMyTasks(tasks)
			tasksChanged = !tasksEqual(s.MyTasks, newTasks)
			s.MyTasks = newTasks
		}
	}

	if tasksChanged {
		// Send PX4 path messages
		if s.MissionInstance > -1 {
			s.MissionInstance++
		}
		sendFlyToMessages(generatePoints(s.MyTasks), pubPath, pubMavlink)
	} else {
		log.Println("Task assigment not changed -> skipping")
	}

	result := make([]types.Message, 0)

	// Leader posts fleets mission plan
	if s.LeaderName == s.MyName {
		result = append(result, s.createMissionPlan())
	}

	// All drones post flight plan
	result = append(result, s.createFlightPlan())

	return result
}

func createMyTasks(tasks []*TaskAssignment) []*myTask {
	mytasks := make([]*myTask, 0)
	for _, t := range tasks {
		path := make([]*taskPoint, 0)
		if t.Type == "fly-to" {
			path = append(path, &taskPoint{Reached: false, X: t.X, Y: t.Y, Z: t.Z})
		} else if t.Type == "execute-preplanned" {
			plan, err := loadPrelanned()
			if err != nil {
				log.Println("Preplanned task skipped due to errors")
				continue
			}
			for _, p := range plan {
				path = append(path, &taskPoint{Reached: false, X: p.X, Y: p.Y, Z: p.Z})
			}
		}
		mytasks = append(mytasks, &myTask{ID: t.ID, Completed: false, Path: path})
	}

	return mytasks
}

func tasksEqual(t1 []*myTask, t2 []*myTask) bool {
	if len(t1) != len(t2) {
		return false
	}

	for i := 0; i < len(t1); i++ {
		if t1[i].ID != t2[i].ID {
			return false
		}
	}

	return true
}

func generatePoints(tasks []*myTask) []rosTypes.Point {
	points := make([]rosTypes.Point, 0)
	for _, t := range tasks {
		for _, p := range t.Path {
			points = append(points, rosTypes.Point{X: p.X, Y: p.Y, Z: p.Z})
		}
	}

	return points
}

func sendFlyToMessages(points []rosTypes.Point, pubPath *ros.Publisher, pubMavlink *ros.Publisher) {
	if len(points) == 0 {
		return
	}

	path := rosTypes.NewPath(points)
	pubPath.DoPublish(path)

	time.Sleep(time.Duration(len(points)*100) * time.Millisecond)

	pubMavlink.DoPublish(rosTypes.GenerateString("start_mission"))
}

func (s *worldState) handleTaskCompleted(msg TaskCompleted) []types.Message {
	for _, bi := range s.Backlog {
		if bi.ID == msg.ID {
			bi.Status = BACKLOG_ITEM_STATUS_COMPLETED
			break
		}
	}

	if s.LeaderName == s.MyName {
		return []types.Message{s.createMissionPlan()}
	}

	return []types.Message{}
}

func (s *worldState) handleMissionResult(msg MissionResult, pubMavlink *ros.Publisher) []types.Message {
	// InstanceCount usually starts from 2 and increments by 1 every time a new path is sent
	if s.MissionInstance == -1 {
		log.Printf("Mission instance count initialized: %v", msg.InstanceCount)
		s.MissionInstance = msg.InstanceCount
	} else if s.MissionInstance != int(msg.InstanceCount) {
		log.Printf("Mission instance count mismatch, expected: %v, given: %v", s.MissionInstance, msg.InstanceCount)
		return []types.Message{}
	}

	// SeqReached: -1, 0, ... N
	// 2 = point-1 reached, 5 = point-2 reached...
	if !(msg.SeqReached > 0 && msg.SeqReached%3 == 2) {
		return []types.Message{}
	}

	pathIndex := msg.SeqReached / 3
	currentIndex := 0
	result := make([]types.Message, 0)
	for ti, t := range s.MyTasks {
		for pi, p := range t.Path {
			if currentIndex <= pathIndex {
				p.Reached = true
				if pi == len(t.Path)-1 {
					if t.Completed == false {
						result = append(result, s.createTaskCompleted(t.ID))
					}
					t.Completed = true
					if ti == len(s.MyTasks)-1 {
						// this is the last task in the drone -> make the drone land
						pubMavlink.DoPublish(rosTypes.GenerateString("land"))
					}
				}
			}

			currentIndex++
		}
	}

	result = append(result, s.createFlightPlan())

	return result
}

func (s *worldState) createTaskCompleted(id string) types.Message {
	msg := serialize(TaskCompleted{id})
	return types.Message{
		Timestamp:   time.Now().UTC(),
		From:        s.MyName,
		To:          "*",
		ID:          "id1",
		MessageType: "task-completed",
		Message:     msg,
	}
}

func (s *worldState) createFlightPlan() types.Message {
	points := make([]*FlightPlan, 0)
	for _, t := range s.MyTasks {
		for _, p := range t.Path {
			points = append(points, &FlightPlan{Reached: p.Reached, X: p.X, Y: p.Y, Z: p.Z})
		}
	}
	msg := serialize(points)
	return types.Message{
		Timestamp:   time.Now().UTC(),
		From:        s.MyName,
		To:          "*",
		ID:          "id1",
		MessageType: "flight-plan",
		Message:     msg,
	}
}

func (s *worldState) createTasksAssigned() []types.Message {
	drones := s.Drones.names()
	result := TasksAssigned{
		Tasks: make(map[string][]*TaskAssignment),
	}
	for i, bi := range s.Backlog {
		if bi.Status == BACKLOG_ITEM_STATUS_COMPLETED {
			continue
		}
		drone := drones[i%len(drones)]
		if bi.Drone != "" {
			drone = bi.Drone
		}
		task := TaskAssignment{
			Type: bi.Type,
			ID:   bi.ID,
			X:    bi.X,
			Y:    bi.Y,
			Z:    bi.Z,
		}

		result.Tasks[drone] = append(result.Tasks[drone], &task)
	}

	msg := serialize(result)
	return []types.Message{
		{
			Timestamp:   time.Now().UTC(),
			From:        s.MyName,
			To:          "*",
			ID:          "id1",
			MessageType: "tasks-assigned",
			Message:     msg,
		},
	}
}

func (s *worldState) createMissionPlan() types.Message {
	result := make([]*MissionPlan, 0)
	for _, bi := range s.Backlog {
		mp := &MissionPlan{
			ID:         bi.ID,
			AssignedTo: "",
			Status:     bi.Status,
		}
		for _, d := range s.Drones {
			for _, t := range d.Tasks {
				if t.ID == bi.ID {
					mp.AssignedTo = d.Name
				}
			}
		}
		result = append(result, mp)
	}

	return types.Message{
		Timestamp:   time.Now().UTC(),
		From:        s.MyName,
		To:          "*",
		ID:          "id1",
		MessageType: "mission-plan",
		Message:     serialize(result),
	}
}

func (drones *Drones) names() []string {
	result := make([]string, 0)
	for k := range *drones {
		result = append(result, k)
	}

	return result
}

func smallestString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	result := values[0]
	for _, v := range values {
		if v < result {
			result = v
		}
	}
	return result
}

func serialize(i interface{}) string {
	b, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}

	return string(b)
}

func deserialize(jsonString string, i interface{}) {
	err := json.Unmarshal([]byte(jsonString), i)
	if err != nil {
		panic(err)
	}
}
