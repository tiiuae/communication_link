package worldengine

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/tiiuae/communication_link/missionengine/types"
)

const (
	BACKLOG_ITEM_STATUS_UNASSIGNED = "unassigned"
	BACKLOG_ITEM_STATUS_ASSIGNED   = "assigned"
	BACKLOG_ITEM_STATUS_QUEUED     = "queued"
	BACKLOG_ITEM_STATUS_STARTED    = "started"
	BACKLOG_ITEM_STATUS_COMPLETED  = "completed"
	BACKLOG_ITEM_STATUS_FAILED     = "failed"
	TASK_STATUS_CREATED            = "created"
	TASK_STATUS_QUEUED             = "queued"
	TASK_STATUS_STARTED            = "started"
	TASK_STATUS_COMPLETED          = "completed"
	TASK_STATUS_FAILED             = "failed"
)

type Drones map[string]*droneState
type Backlog []*backlogItem

type worldState struct {
	// My data
	MyName  string
	MyTasks []*myTask

	// Fleet data
	Backlog Backlog
	Drones  Drones
}

type droneState struct {
	Name string
}

type myTask struct {
	ID         string
	Status     string
	InstanceID int
	Path       []*taskPoint
}

type taskPoint struct {
	Reached bool
	X       float64
	Y       float64
	Z       float64
}

type backlogItem struct {
	// Task definition
	ID    string
	Type  string
	Drone string
	X     float64
	Y     float64
	Z     float64
	// Runtime status
	Status     string
	AssignedTo string
}

func createState(me string) *worldState {
	return &worldState{
		MyName:  me,
		MyTasks: make([]*myTask, 0),
		Backlog: make([]*backlogItem, 0),
		Drones:  make(map[string]*droneState, 0),
	}
}

func (s *worldState) handleDroneAdded(msg DroneAdded) []types.MessageOut {
	name := msg.Name
	s.Drones[name] = &droneState{name}

	return []types.MessageOut{}
}

func (s *worldState) handleDroneRemoved(msg DroneRemoved) []types.MessageOut {
	delete(s.Drones, msg.Name)

	return []types.MessageOut{}
}

func (s *worldState) handleFlyToTaskCreated(msg FlyToTaskCreated) []types.MessageOut {
	// Add task to backlog
	bi := backlogItem{
		ID:     msg.ID,
		Status: BACKLOG_ITEM_STATUS_UNASSIGNED,
		Type:   msg.Type,
		Drone:  "",
		X:      msg.Payload.X,
		Y:      msg.Payload.Y,
		Z:      msg.Payload.Z,
	}
	s.Backlog = append(s.Backlog, &bi)

	return []types.MessageOut{}
}

func (s *worldState) handleExecutePredefinedToTaskCreated(msg ExecutePredefinedTaskCreated) []types.MessageOut {
	// Add task to backlog
	bi := backlogItem{
		ID:     msg.ID,
		Status: BACKLOG_ITEM_STATUS_UNASSIGNED,
		Type:   msg.Type,
		Drone:  msg.Payload.Drone,
		X:      0,
		Y:      0,
		Z:      0,
	}
	s.Backlog = append(s.Backlog, &bi)

	return []types.MessageOut{}
}

func (s *worldState) handleTasksAssigned(msg TasksAssigned) []types.MessageOut {
	for droneName, tasks := range msg.Tasks {
		for _, task := range tasks {
			bi, found := s.getBacklogItem(task.ID)
			if found && bi.Status == BACKLOG_ITEM_STATUS_UNASSIGNED {
				bi.Status = BACKLOG_ITEM_STATUS_ASSIGNED
				bi.AssignedTo = droneName
			}
			if droneName != s.MyName {
				continue
			}
			_, found = s.getTask(task.ID)
			if found {
				continue
			}
			newTask, err := createMyTask(task, s.MyName)
			if err != nil {
				log.Print(err)
				continue
			}
			s.MyTasks = append(s.MyTasks, newTask)
		}
	}

	// Leader posts fleets mission plan
	if s.isLeader() {
		return []types.MessageOut{s.createMissionPlan()}
	}

	return []types.MessageOut{}
}

func (s *worldState) processTasks() []types.MessageOut {
	result := make([]types.MessageOut, 0)
	result = append(result, s.queueTasks()...)

	queued := s.getCurrentTask()
	if queued == nil {
		log.Println("Processing tasks: Task assigment not changed -> skipping")
		return result
	}

	if queued.Status == TASK_STATUS_QUEUED {
		queued.Status = TASK_STATUS_STARTED
		result = append(result, s.createFlyToMessages(generatePoints(queued))...)
		result = append(result, s.createFlightPlan())
		result = append(result, s.createTaskStarted(queued.ID))
	}

	return result
}

func (s *worldState) handleTaskQueued(msg TaskCompleted) []types.MessageOut {
	for _, bi := range s.Backlog {
		if bi.ID == msg.ID {
			bi.Status = BACKLOG_ITEM_STATUS_QUEUED
			break
		}
	}

	if s.isLeader() {
		return []types.MessageOut{s.createMissionPlan()}
	}

	return []types.MessageOut{}
}

func (s *worldState) handleTaskStarted(msg TaskCompleted) []types.MessageOut {
	for _, bi := range s.Backlog {
		if bi.ID == msg.ID {
			bi.Status = BACKLOG_ITEM_STATUS_STARTED
			break
		}
	}

	if s.isLeader() {
		return []types.MessageOut{s.createMissionPlan()}
	}

	return []types.MessageOut{}
}

func (s *worldState) handleTaskCompleted(msg TaskCompleted) []types.MessageOut {
	for _, bi := range s.Backlog {
		if bi.ID == msg.ID {
			bi.Status = BACKLOG_ITEM_STATUS_COMPLETED
			break
		}
	}

	if s.isLeader() {
		return []types.MessageOut{s.createMissionPlan()}
	}

	return []types.MessageOut{}
}

func (s *worldState) handleTaskFailed(msg TaskCompleted) []types.MessageOut {
	for _, bi := range s.Backlog {
		if bi.ID == msg.ID {
			bi.Status = BACKLOG_ITEM_STATUS_FAILED
			break
		}
	}

	if s.isLeader() {
		return []types.MessageOut{s.createMissionPlan()}
	}

	return []types.MessageOut{}
}

var missionResultFilter string = ""

func (s *worldState) handleMissionResult(msg MissionResult) []types.MessageOut {
	key := fmt.Sprintf("%v-%v-%v-%v", msg.InstanceCount, msg.SeqReached, msg.Valid, msg.Finished)
	if key == missionResultFilter {
		log.Print("MissionResult: duplicate filter")
		return []types.MessageOut{}
	}
	missionResultFilter = key

	// InstanceCount
	// 	- usually starts from 1 when no path is yet sent
	//  - increments by 1 every time a new path is sent
	// 	- sometimes increments randomly by itself
	//	  -> doesn't match to any path sent, but SeqReached seems to be -1 always
	task := s.getTaskByInstanceCount(msg.InstanceCount)
	if task == nil {
		log.Printf("MissionResult: no matching task found for instance count: %d", msg.InstanceCount)
		return []types.MessageOut{}
	}
	if task.InstanceID == -1 {
		log.Printf("MissionResult: matching instance count: %d to task %s", msg.InstanceCount, task.ID)
		task.InstanceID = msg.InstanceCount
	}

	if msg.Valid == false {
		task.Status = TASK_STATUS_FAILED
		return []types.MessageOut{s.createTaskFailed(task.ID)}
	}

	// SeqReached: -1, 0, ... N
	// Continue if SeqReached is: 2 (point-1 reached), 5 (point-2 reached)...
	if msg.SeqReached%3 != 2 {
		return []types.MessageOut{}
	}

	pathIndex := msg.SeqReached / 3
	result := make([]types.MessageOut, 0)
	for pi, p := range task.Path {
		if pi <= pathIndex {
			p.Reached = true
			if pi == len(task.Path)-1 {
				if task.Status != TASK_STATUS_COMPLETED {
					result = append(result, s.createTaskCompleted(task.ID))
				}
				task.Status = TASK_STATUS_COMPLETED
				nextTask := s.getCurrentTask()
				if nextTask == nil {
					// Last task on my backlog -> land the drone
					result = append(result, s.createLandMessage())
				}
			}
		}
	}

	if msg.Finished == true && task.Status != TASK_STATUS_COMPLETED {
		panic("TODO: MissionResult mismatch")
	}

	result = append(result, s.createFlightPlan())

	return result
}

func (s *worldState) process() []types.MessageOut {
	if !s.isLeader() {
		return []types.MessageOut{}
	}

	return s.createTasksAssigned()
}

func createMyTask(t *TaskAssignment, droneName string) (*myTask, error) {
	path := make([]*taskPoint, 0)
	if t.Type == "fly-to" {
		path = append(path, &taskPoint{Reached: false, X: t.X, Y: t.Y, Z: t.Z})
	} else if t.Type == "execute-preplanned" {
		plan, err := loadPrelanned(droneName)
		if err != nil {
			return nil, errors.WithMessage(err, "Preplanned task skipped due to errors")
		}
		for _, p := range plan {
			path = append(path, &taskPoint{Reached: false, X: p.X, Y: p.Y, Z: p.Z})
		}
	}

	return &myTask{
		ID:         t.ID,
		Status:     TASK_STATUS_CREATED,
		InstanceID: -1,
		Path:       path,
	}, nil
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

func generatePoints(task *myTask) []Point {
	points := make([]Point, 0)
	for _, p := range task.Path {
		points = append(points, Point{X: p.X, Y: p.Y, Z: p.Z})
	}

	return points
}

func (s *worldState) createFlyToMessages(points []Point) []types.MessageOut {
	return []types.MessageOut{
		{
			Timestamp:   time.Now().UTC(),
			From:        s.MyName,
			To:          s.MyName,
			ID:          "id1",
			MessageType: "flight-path",
			Message:     FlightPath{points},
		},
		{
			Timestamp:   time.Now().UTC(),
			From:        s.MyName,
			To:          s.MyName,
			ID:          "id1",
			MessageType: "start-mission",
			Message:     StartMission{Delay: time.Duration(len(points)*100) * time.Millisecond},
		},
	}
}

func (s *worldState) createLandMessage() types.MessageOut {
	return types.MessageOut{
		Timestamp:   time.Now().UTC(),
		From:        s.MyName,
		To:          s.MyName,
		ID:          "id1",
		MessageType: "land",
		Message:     Land{},
	}
}

func (s *worldState) queueTasks() []types.MessageOut {
	result := make([]types.MessageOut, 0)
	for _, x := range s.MyTasks {
		if x.Status == TASK_STATUS_CREATED {
			x.Status = TASK_STATUS_QUEUED
			result = append(result, s.createTaskQueued(x.ID))
		}
	}

	return result
}

func (s *worldState) createTaskQueued(id string) types.MessageOut {
	return types.MessageOut{
		Timestamp:   time.Now().UTC(),
		From:        s.MyName,
		To:          "*",
		ID:          "id1",
		MessageType: "task-queued",
		Message:     TaskQueued{id},
	}
}

func (s *worldState) createTaskStarted(id string) types.MessageOut {
	return types.MessageOut{
		Timestamp:   time.Now().UTC(),
		From:        s.MyName,
		To:          "*",
		ID:          "id1",
		MessageType: "task-started",
		Message:     TaskStarted{id},
	}
}

func (s *worldState) createTaskCompleted(id string) types.MessageOut {
	return types.MessageOut{
		Timestamp:   time.Now().UTC(),
		From:        s.MyName,
		To:          "*",
		ID:          "id1",
		MessageType: "task-completed",
		Message:     TaskCompleted{id},
	}
}

func (s *worldState) createTaskFailed(id string) types.MessageOut {
	return types.MessageOut{
		Timestamp:   time.Now().UTC(),
		From:        s.MyName,
		To:          "*",
		ID:          "id1",
		MessageType: "task-failed",
		Message:     TaskFailed{id},
	}
}

func (s *worldState) createFlightPlan() types.MessageOut {
	points := make([]*FlightPlan, 0)
	for _, t := range s.MyTasks {
		if t.Status == TASK_STATUS_FAILED {
			// Skip failed tasks
			continue
		}
		for _, p := range t.Path {
			points = append(points, &FlightPlan{Reached: p.Reached, X: p.X, Y: p.Y, Z: p.Z})
		}
	}
	return types.MessageOut{
		Timestamp:   time.Now().UTC(),
		From:        s.MyName,
		To:          "*",
		ID:          "id1",
		MessageType: "flight-plan",
		Message:     points,
	}
}

func (s *worldState) createTasksAssigned() []types.MessageOut {
	drones := s.Drones.names()
	result := TasksAssigned{
		Tasks: make(map[string][]*TaskAssignment),
	}
	for i, bi := range s.Backlog {
		// Skip completed tasks
		if bi.Status == BACKLOG_ITEM_STATUS_COMPLETED {
			continue
		}

		// Assign unassigned tasks
		if bi.Status == BACKLOG_ITEM_STATUS_UNASSIGNED {
			if bi.Drone != "" {
				// Target drone from task parameters
				bi.AssignedTo = bi.Drone
			} else {
				// Random selection
				bi.AssignedTo = drones[i%len(drones)]
			}

			bi.Status = BACKLOG_ITEM_STATUS_ASSIGNED
		}

		task := TaskAssignment{
			Type: bi.Type,
			ID:   bi.ID,
			X:    bi.X,
			Y:    bi.Y,
			Z:    bi.Z,
		}

		result.Tasks[bi.AssignedTo] = append(result.Tasks[bi.AssignedTo], &task)
	}

	if len(result.Tasks) == 0 {
		return []types.MessageOut{}
	}

	return []types.MessageOut{
		{
			Timestamp:   time.Now().UTC(),
			From:        s.MyName,
			To:          "*",
			ID:          "id1",
			MessageType: "tasks-assigned",
			Message:     result,
		},
	}
}

func (s *worldState) createMissionPlan() types.MessageOut {
	result := make([]*MissionPlan, 0)
	for _, bi := range s.Backlog {
		mp := &MissionPlan{
			ID:         bi.ID,
			AssignedTo: bi.AssignedTo,
			Status:     bi.Status,
		}

		result = append(result, mp)
	}

	return types.MessageOut{
		Timestamp:   time.Now().UTC(),
		From:        s.MyName,
		To:          "*",
		ID:          "id1",
		MessageType: "mission-plan",
		Message:     result,
	}
}

func (drones *Drones) names() []string {
	result := make([]string, 0)
	for k := range *drones {
		result = append(result, k)
	}

	sort.Strings(result)
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

func (s *worldState) isLeader() bool {
	if len(s.Drones) == 0 {
		return false
	}

	keys := make([]string, 0)
	for k := range s.Drones {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys[0] == s.MyName
}

func (s *worldState) getTask(ID string) (*myTask, bool) {
	for _, x := range s.MyTasks {
		if x.ID == ID {
			return x, true
		}
	}

	return nil, false
}

func (s *worldState) getBacklogItem(ID string) (*backlogItem, bool) {
	for _, x := range s.Backlog {
		if x.ID == ID {
			return x, true
		}
	}

	return nil, false
}

func (s *worldState) getCurrentTask() *myTask {
	for _, task := range s.MyTasks {
		if task.Status != TASK_STATUS_COMPLETED {
			return task
		}
	}

	return nil
}

func (s *worldState) getTaskByInstanceCount(instanceCount int) *myTask {
	if len(s.MyTasks) == 0 {
		return nil
	}

	for _, t := range s.MyTasks {
		if t.InstanceID == instanceCount {
			// Match found
			return t
		}
		if t.InstanceID == -1 && t.Status == TASK_STATUS_STARTED {
			// First unmatched inprogress task
			return t
		}
	}

	return nil
}
