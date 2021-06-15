package missionplanner

import (
	"log"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/tiiuae/communication_link/missionengine/internal/types"
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

type state struct {
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
	ID     string
	Status string
	Path   []*taskPoint
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

func createState(me string) *state {
	return &state{
		MyName:  me,
		MyTasks: make([]*myTask, 0),
		Backlog: make([]*backlogItem, 0),
		Drones:  make(map[string]*droneState, 0),
	}
}

func (s *state) handleDroneAdded(msg types.DroneAdded) []types.Message {
	name := msg.Name
	s.Drones[name] = &droneState{name}

	return []types.Message{}
}

func (s *state) handleDroneRemoved(msg types.DroneRemoved) []types.Message {
	delete(s.Drones, msg.Name)

	return []types.Message{}
}

func (s *state) handleFlyToTaskCreated(msg types.FlyToTaskCreated) []types.Message {
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

	return []types.Message{}
}

func (s *state) handleExecutePredefinedToTaskCreated(msg types.ExecutePredefinedTaskCreated) []types.Message {
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

	return []types.Message{}
}

func (s *state) handleTasksAssigned(msg types.TasksAssigned) []types.Message {
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
		return []types.Message{s.createMissionPlan()}
	}

	return []types.Message{}
}

func (s *state) processTasks() []types.Message {
	result := make([]types.Message, 0)
	result = append(result, s.queueTasks()...)

	queued := s.getCurrentTask()
	if queued == nil {
		log.Println("Processing tasks: Task assigment not changed -> skipping")
		return result
	}

	if queued.Status == TASK_STATUS_QUEUED {
		queued.Status = TASK_STATUS_STARTED
		result = append(result, s.createFlyToMessages(queued.ID, generatePoints(queued))...)
		result = append(result, s.createFlightPlan())
		result = append(result, s.createTaskStarted(queued.ID))
	}

	return result
}

func (s *state) handleTaskQueued(msg types.TaskQueued) []types.Message {
	for _, bi := range s.Backlog {
		if bi.ID == msg.ID {
			bi.Status = BACKLOG_ITEM_STATUS_QUEUED
			break
		}
	}

	if s.isLeader() {
		return []types.Message{s.createMissionPlan()}
	}

	return []types.Message{}
}

func (s *state) handleTaskStarted(msg types.TaskStarted) []types.Message {
	for _, bi := range s.Backlog {
		if bi.ID == msg.ID {
			bi.Status = BACKLOG_ITEM_STATUS_STARTED
			break
		}
	}

	if s.isLeader() {
		return []types.Message{s.createMissionPlan()}
	}

	return []types.Message{}
}

func (s *state) handleTaskCompleted(msg types.TaskCompleted) []types.Message {
	for _, bi := range s.Backlog {
		if bi.ID == msg.ID {
			bi.Status = BACKLOG_ITEM_STATUS_COMPLETED
			break
		}
	}

	if s.isLeader() {
		return []types.Message{s.createMissionPlan()}
	}

	return []types.Message{}
}

func (s *state) handleTaskFailed(msg types.TaskFailed) []types.Message {
	for _, bi := range s.Backlog {
		if bi.ID == msg.ID {
			bi.Status = BACKLOG_ITEM_STATUS_FAILED
			break
		}
	}

	if s.isLeader() {
		return []types.Message{s.createMissionPlan()}
	}

	return []types.Message{}
}

func (s *state) handleFlyPathProgress(msg types.FlyPathProgress) []types.Message {
	task, found := s.getTask(msg.ID)
	if !found {
		log.Printf("handleFlyPathProgress: task '%s' not found", msg.ID)
		return []types.Message{}
	}

	if msg.Failed {
		task.Status = TASK_STATUS_FAILED
		return []types.Message{s.createTaskFailed(task.ID)}
	}

	if msg.PathCompleted {
		for _, x := range task.Path {
			x.Reached = true
		}
		task.Status = TASK_STATUS_COMPLETED
		result := make([]types.Message, 0)
		result = append(result, s.createTaskCompleted(task.ID))
		nextTask := s.getCurrentTask()
		if nextTask == nil {
			// Last task on my backlog -> land the drone
			result = append(result, s.createLandMessage())
		}
		result = append(result, s.createFlightPlan())

		return result
	}

	return []types.Message{}
}

func (s *state) process() []types.Message {
	if !s.isLeader() {
		return []types.Message{}
	}

	return s.createTasksAssigned()
}

func createMyTask(t *types.TaskAssignment, droneName string) (*myTask, error) {
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
		ID:     t.ID,
		Status: TASK_STATUS_CREATED,
		Path:   path,
	}, nil
}

func generatePoints(task *myTask) []types.Point {
	points := make([]types.Point, 0)
	for _, p := range task.Path {
		points = append(points, types.Point{X: p.X, Y: p.Y, Z: p.Z})
	}

	return points
}

func (s *state) createFlyToMessages(id string, points []types.Point) []types.Message {
	return []types.Message{
		{
			Timestamp:   time.Now().UTC(),
			From:        s.MyName,
			To:          s.MyName,
			ID:          "id1",
			MessageType: "fly-path",
			Message:     types.FlyPath{id, points},
		},
	}
}

func (s *state) createLandMessage() types.Message {
	return types.Message{
		Timestamp:   time.Now().UTC(),
		From:        s.MyName,
		To:          s.MyName,
		ID:          "id1",
		MessageType: "land",
		Message:     types.Land{},
	}
}

func (s *state) queueTasks() []types.Message {
	result := make([]types.Message, 0)
	for _, x := range s.MyTasks {
		if x.Status == TASK_STATUS_CREATED {
			x.Status = TASK_STATUS_QUEUED
			result = append(result, s.createTaskQueued(x.ID))
		}
	}

	return result
}

func (s *state) createTaskQueued(id string) types.Message {
	return types.Message{
		Timestamp:   time.Now().UTC(),
		From:        s.MyName,
		To:          "*",
		ID:          "id1",
		MessageType: "task-queued",
		Message:     types.TaskQueued{id},
	}
}

func (s *state) createTaskStarted(id string) types.Message {
	return types.Message{
		Timestamp:   time.Now().UTC(),
		From:        s.MyName,
		To:          "*",
		ID:          "id1",
		MessageType: "task-started",
		Message:     types.TaskStarted{id},
	}
}

func (s *state) createTaskCompleted(id string) types.Message {
	return types.Message{
		Timestamp:   time.Now().UTC(),
		From:        s.MyName,
		To:          "*",
		ID:          "id1",
		MessageType: "task-completed",
		Message:     types.TaskCompleted{id},
	}
}

func (s *state) createTaskFailed(id string) types.Message {
	return types.Message{
		Timestamp:   time.Now().UTC(),
		From:        s.MyName,
		To:          "*",
		ID:          "id1",
		MessageType: "task-failed",
		Message:     types.TaskFailed{id},
	}
}

func (s *state) createFlightPlan() types.Message {
	points := make([]*types.FlightPlan, 0)
	for _, t := range s.MyTasks {
		if t.Status == TASK_STATUS_FAILED {
			// Skip failed tasks
			continue
		}
		for _, p := range t.Path {
			points = append(points, &types.FlightPlan{Reached: p.Reached, X: p.X, Y: p.Y, Z: p.Z})
		}
	}
	return types.Message{
		Timestamp:   time.Now().UTC(),
		From:        s.MyName,
		To:          "*",
		ID:          "id1",
		MessageType: "flight-plan",
		Message:     points,
	}
}

func (s *state) createTasksAssigned() []types.Message {
	drones := s.Drones.names()
	result := types.TasksAssigned{
		Tasks: make(map[string][]*types.TaskAssignment),
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

		task := types.TaskAssignment{
			Type: bi.Type,
			ID:   bi.ID,
			X:    bi.X,
			Y:    bi.Y,
			Z:    bi.Z,
		}

		result.Tasks[bi.AssignedTo] = append(result.Tasks[bi.AssignedTo], &task)
	}

	if len(result.Tasks) == 0 {
		return []types.Message{}
	}

	return []types.Message{
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

func (s *state) createMissionPlan() types.Message {
	result := make([]*types.MissionPlan, 0)
	for _, bi := range s.Backlog {
		mp := &types.MissionPlan{
			ID:         bi.ID,
			AssignedTo: bi.AssignedTo,
			Status:     bi.Status,
		}

		result = append(result, mp)
	}

	return types.Message{
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

func (s *state) isLeader() bool {
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

func (s *state) getTask(ID string) (*myTask, bool) {
	for _, x := range s.MyTasks {
		if x.ID == ID {
			return x, true
		}
	}

	return nil, false
}

func (s *state) getBacklogItem(ID string) (*backlogItem, bool) {
	for _, x := range s.Backlog {
		if x.ID == ID {
			return x, true
		}
	}

	return nil, false
}

func (s *state) getCurrentTask() *myTask {
	for _, task := range s.MyTasks {
		if task.Status != TASK_STATUS_COMPLETED {
			return task
		}
	}

	return nil
}
