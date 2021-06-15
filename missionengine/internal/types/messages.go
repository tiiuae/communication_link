package types

type JoinMission struct {
	GitServer    string
	GitServerKey string
	MissionSlug  string
}

type UpdateBacklog struct{}

type LeaveMission struct{}

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

type TaskQueued struct {
	ID string `json:"id"`
}

type TaskStarted struct {
	ID string `json:"id"`
}

type TaskCompleted struct {
	ID string `json:"id"`
}

type TaskFailed struct {
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

type FlyPath struct {
	ID     string  `json:"id"`
	Points []Point `json:"points"`
}

type FlyPathProgress struct {
	ID            string `json:"id"`
	Index         int    `json:"point"`
	PathCompleted bool   `json:"path_completed"`
	Failed        bool   `json:"failed"`
}

type SendPath struct {
	Points []Point `json:"points"`
}

type StartMission struct {
}

type Land struct {
}

type JoinedMission struct {
	MissionSlug string `json:"mission_slug"`
}

type LeftMission struct {
	MissionSlug string `json:"mission_slug"`
}

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

type Process struct{}

type VehicleState struct {
	ArmingState     ArmingState
	NavigationState NavigationState
}

type ArmingState uint8

const (
	ArmingStateInit0 ArmingState = iota
	ArmingStateStandBy1
	ArmingStateArmed2
	ArmingStateStandByError3
	ArmingStateShutdown4
	ArmingStateInAirRestore5
)

type NavigationState uint8

const (
	NavigationStateManualMode0 NavigationState = iota
	NavigationStateAltitudeControlMode1
	NavigationStatePositionControlMode2
	NavigationStateAutoMissionMode3
	NavigationStateAutoLoiterMode4
	NavigationStateAutoReturnToLaunchMode5
	NavigationStateUnknown6
	NavigationStateUnknown7
	NavigationStateAutoLandOnEngineFailure8
	NavigationStateAutoLandOnGpsFailure9
	NavigationStateAcroMode10
	NavigationStateUnknown11
	NavigationStateDescendMode12
	NavigationStateTerminationMode13
	NavigationStateOffboardMode14
	NavigationStateStabilizedMode15
	NavigationStateRattitudeFlipMode16
	NavigationStateTakeoffMode17
	NavigationStateLandMode18
	NavigationStateFollowMode19
	NavigationStatePrecisionLandWithLandingTarget20
	NavigationStateOrbitMode21
)

type GlobalPosition struct {
	Lat float64
	Lon float64
	Alt float64
}
