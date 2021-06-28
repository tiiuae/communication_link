package flyf4f

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tiiuae/communication_link/missionengine/internal/types"
	"github.com/tiiuae/rclgo/pkg/ros2"
	builtin_interfaces "github.com/tiiuae/rclgo/pkg/ros2/msgs/builtin_interfaces/msg"
	fog_msgs "github.com/tiiuae/rclgo/pkg/ros2/msgs/fog_msgs/srv"
	geometry_msgs "github.com/tiiuae/rclgo/pkg/ros2/msgs/geometry_msgs/msg"
	nav_msgs "github.com/tiiuae/rclgo/pkg/ros2/msgs/nav_msgs/msg"
	std_msgs "github.com/tiiuae/rclgo/pkg/ros2/msgs/std_msgs/msg"
	std_srvs "github.com/tiiuae/rclgo/pkg/ros2/msgs/std_srvs/srv"

	"github.com/tiiuae/rclgo/pkg/ros2/ros2types"
)

type f4f struct {
	ctx        context.Context
	rclContext *ros2.Context
	node       *ros2.Node
	deviceID   string
	inbox      chan types.Message
	state      *state
}

func New(ctx context.Context, rclContext *ros2.Context, node *ros2.Node, deviceID string) types.MessageHandler {
	return &f4f{ctx, rclContext, node, deviceID, make(chan types.Message, 10), newState()}
}

func (f4f *f4f) Run(ctx context.Context, wg *sync.WaitGroup, post types.PostFn) {
	go f4f.runMessageLoop(ctx, wg, post)
	go f4f.runF4FSubscriber(ctx, wg)
}

func (f4f *f4f) Receive(message types.Message) {
	f4f.inbox <- message
}

func (f4f *f4f) runMessageLoop(ctx context.Context, wg *sync.WaitGroup, post types.PostFn) {
	wg.Add(1)
	defer wg.Done()

	armingService := createArmingService(f4f.ctx, f4f.rclContext, f4f.node)
	takeoffService := createTakeoffService(f4f.ctx, f4f.rclContext, f4f.node)
	landingService := createLandingService(f4f.ctx, f4f.rclContext, f4f.node)
	waypointService := createWaypointService(f4f.ctx, f4f.rclContext, f4f.node)

	defer armingService.Close()
	defer takeoffService.Close()
	defer waypointService.Close()

	for {
		select {
		case <-ctx.Done():
			log.Println("F4F shutting down")
			return
		case msg := <-f4f.inbox:
			switch m := msg.Message.(type) {
			case types.VehicleState:
				f4f.state.handleVehicleState(m)
				publishMessages(ctx, armingService, takeoffService, waypointService, landingService, post, f4f.state.process())
			case NavigationStatus:
				f4f.state.handleNavigationStatus(m)
				publishMessages(ctx, armingService, takeoffService, waypointService, landingService, post, f4f.state.process())
			case types.FlyPath:
				f4f.state.handleFlyPath(m)
				publishMessages(ctx, armingService, takeoffService, waypointService, landingService, post, f4f.state.process())
			case types.Land:
				publishMessages(ctx, armingService, takeoffService, waypointService, landingService, post, []types.Message{msg})
			}
		}
	}
}

func publishMessages(ctx context.Context, armingService *ros2.Client, takeoffService *ros2.Client, waypointService *ros2.Client, landingService *ros2.Client, post types.PostFn, messages []types.Message) {
	for _, msg := range messages {
		switch m := msg.Message.(type) {
		case Arm:
			req := std_srvs.NewSetBool_Request()
			req.Data = true
			res, _, err := armingService.Send(ctx, req)
			if err != nil {
				log.Fatalf("%v", err)
			}
			log.Printf("F4F: ARMING: %v", res)
		case TakeOff:
			req := std_srvs.NewTrigger_Request()
			res, _, err := takeoffService.Send(ctx, req)
			if err != nil {
				log.Fatalf("%v", err)
			}
			log.Printf("F4F: TAKEOFF: %v", res)
		case types.Land:
			req := std_srvs.NewTrigger_Request()
			res, _, err := landingService.Send(ctx, req)
			if err != nil {
				log.Fatalf("%v", err)
			}
			log.Printf("F4F: LAND: %v", res)
		case types.SendPath:
			req := fog_msgs.NewVec4_Request()
			req.Goal = []float64{m.Points[0].X, m.Points[0].Y, m.Points[0].Z, 0.0}
			res, _, err := waypointService.Send(ctx, req)
			if err != nil {
				log.Fatalf("%v", err)
			}
			log.Printf("F4F: WAYPOINT_IN: %v -> %v", req.Goal, res)
		case types.FlyPathProgress:
			post(msg)
		}
	}
}

func (f4f *f4f) runF4FSubscriber(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	var previousStatus string = ""

	sub, rclErr := f4f.node.NewSubscription("navigation/status_out", &std_msgs.String{}, func(s *ros2.Subscription) {
		var m std_msgs.String
		_, rlcErr := s.TakeMessage(&m)
		if rlcErr != nil {
			log.Print("TakeMessage failed: runF4FSubscriber")
			return
		}

		currentStatus := fmt.Sprintf("%v", m.Data)

		if currentStatus == previousStatus {
			return
		}

		previousStatus = currentStatus

		out := types.Message{
			Timestamp:   time.Now().UTC(),
			From:        f4f.deviceID,
			To:          f4f.deviceID,
			ID:          "id-1",
			MessageType: "navigation-status",
			Message: NavigationStatus{
				Status: currentStatus,
			},
		}

		log.Printf("F4F: NavigationStatus: %s", currentStatus)

		f4f.inbox <- out
	})

	if rclErr != nil {
		log.Fatalf("Unable to subscribe to topic 'status_out': %v", rclErr)
	}

	err := sub.Spin(ctx, 5*time.Second)
	if err != nil {
		log.Printf("Subscription failed: %v", err)
	}
}

func createTakeoffService(ctx context.Context, rclContext *ros2.Context, node *ros2.Node) *ros2.Client {
	opt := &ros2.ClientOptions{Qos: ros2.NewRmwQosProfileServicesDefault()}
	client, err := node.NewClient("control_interface/takeoff", std_srvs.Trigger, opt)
	if err != nil {
		log.Fatalf("%v", err)
	}

	ws, err := rclContext.NewWaitSet(200 * time.Millisecond)
	if err != nil {
		log.Fatalf("%v", err)
	}

	ws.AddClients(client)
	ws.RunGoroutine(ctx)

	return client
}

func createLandingService(ctx context.Context, rclContext *ros2.Context, node *ros2.Node) *ros2.Client {
	opt := &ros2.ClientOptions{Qos: ros2.NewRmwQosProfileServicesDefault()}
	client, err := node.NewClient("control_interface/land", std_srvs.Trigger, opt)
	if err != nil {
		log.Fatalf("%v", err)
	}

	ws, err := rclContext.NewWaitSet(200 * time.Millisecond)
	if err != nil {
		log.Fatalf("%v", err)
	}

	ws.AddClients(client)
	ws.RunGoroutine(ctx)

	return client
}

func createArmingService(ctx context.Context, rclContext *ros2.Context, node *ros2.Node) *ros2.Client {
	opt := &ros2.ClientOptions{Qos: ros2.NewRmwQosProfileServicesDefault()}
	client, err := node.NewClient("control_interface/arming", std_srvs.SetBool, opt)
	if err != nil {
		log.Fatalf("%v", err)
	}

	ws, err := rclContext.NewWaitSet(200 * time.Millisecond)
	if err != nil {
		log.Fatalf("%v", err)
	}

	ws.AddClients(client)
	ws.RunGoroutine(ctx)

	return client
}

func createWaypointService(ctx context.Context, rclContext *ros2.Context, node *ros2.Node) *ros2.Client {
	opt := &ros2.ClientOptions{Qos: ros2.NewRmwQosProfileServicesDefault()}
	client, err := node.NewClient("navigation/gps_waypoint", fog_msgs.Vec4, opt)
	if err != nil {
		log.Fatalf("%v", err)
	}

	ws, err := rclContext.NewWaitSet(200 * time.Millisecond)
	if err != nil {
		log.Fatalf("%v", err)
	}

	ws.AddClients(client)
	ws.RunGoroutine(ctx)

	return client
}

func createString(value string) ros2types.ROS2Msg {
	rosmsg := std_msgs.NewString()
	rosmsg.Data.SetDefaults(value)
	return rosmsg
}

func createPath(points []types.Point) *nav_msgs.Path {
	path := nav_msgs.NewPath()
	path.Header = *std_msgs.NewHeader()
	path.Header.Stamp = *builtin_interfaces.NewTime()
	path.Header.Stamp.Sec = 10000
	path.Header.Stamp.Nanosec = 100000
	path.Header.FrameId = "map"
	path.Poses = make([]geometry_msgs.PoseStamped, len(points))
	for i, p := range points {
		point := geometry_msgs.NewPoint()
		point.X = p.X
		point.Y = p.Y
		point.Z = p.Z
		pose := geometry_msgs.NewPoseStamped()
		pose.Header = *std_msgs.NewHeader()
		pose.Header.Stamp = *builtin_interfaces.NewTime()
		pose.Header.Stamp.Sec = 10000
		pose.Header.Stamp.Nanosec = 100000
		pose.Header.FrameId = "map"
		pose.Pose.Position = *point
		path.Poses[i] = *pose
	}

	return path
}

type NavigationStatus struct {
	Status string
}

type Arm struct{}
type TakeOff struct{}
