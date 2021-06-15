package telemetry

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/tiiuae/communication_link/missionengine/internal/types"
	"github.com/tiiuae/rclgo/pkg/ros2"
	px4_msgs "github.com/tiiuae/rclgo/pkg/ros2/msgs/px4_msgs/msg"
)

type telemetry struct {
	node     *ros2.Node
	deviceID string
}

func New(node *ros2.Node, deviceID string) types.MessageHandler {
	return &telemetry{node, deviceID}
}

func (t *telemetry) Run(ctx context.Context, wg *sync.WaitGroup, post types.PostFn) {
	go t.runVehicleStatusSubscriber(ctx, wg, post)
	go t.runVehiclePositionSubscriber(ctx, wg, post)
}

func (t *telemetry) Receive(message types.Message) {
}

func (t *telemetry) runVehicleStatusSubscriber(ctx context.Context, wg *sync.WaitGroup, post types.PostFn) {
	wg.Add(1)
	defer wg.Done()

	var currentArmingState uint8
	var currentNavState uint8
	sub, rclErr := t.node.NewSubscription("VehicleStatus_PubSubTopic", &px4_msgs.VehicleStatus{}, func(s *ros2.Subscription) {
		var m px4_msgs.VehicleStatus
		_, rlcErr := s.TakeMessage(&m)
		if rlcErr != nil {
			log.Print("TakeMessage failed: runVehicleStatusSubscriber")
			return
		}

		if currentArmingState == m.ArmingState && currentNavState == m.NavState {
			return
		}

		currentArmingState = m.ArmingState
		currentNavState = m.NavState

		out := types.VehicleState{ArmingState: types.ArmingState(m.ArmingState), NavigationState: types.NavigationState(m.NavState)}
		post(types.CreateMessage("vehicle-status", t.deviceID, t.deviceID, out))
	})

	if rclErr != nil {
		log.Fatalf("Unable to subscribe to topic 'VehicleStatus_PubSubTopic': %v", rclErr)
	}

	err := sub.Spin(ctx, 5*time.Second)
	if err != nil {
		log.Printf("Subscription failed: %v", err)
	}
}

func (t *telemetry) runVehiclePositionSubscriber(ctx context.Context, wg *sync.WaitGroup, post types.PostFn) {
	wg.Add(1)
	defer wg.Done()

	sub, rclErr := t.node.NewSubscription("VehicleGlobalPosition_PubSubTopic", &px4_msgs.VehicleGlobalPosition{}, func(s *ros2.Subscription) {
		var m px4_msgs.VehicleGlobalPosition
		_, rlcErr := s.TakeMessage(&m)
		if rlcErr != nil {
			log.Print("TakeMessage failed: runVehiclePositionSubscriber")
			return
		}

		out := types.GlobalPosition{Lat: m.Lat, Lon: m.Lon, Alt: float64(m.Alt)}
		post(types.CreateMessage("global-position", t.deviceID, t.deviceID, out))
	})

	if rclErr != nil {
		log.Fatalf("Unable to subscribe to topic 'VehicleGlobalPosition_PubSubTopic': %v", rclErr)
	}

	err := sub.Spin(ctx, 5*time.Second)
	if err != nil {
		log.Printf("Subscription failed: %v", err)
	}
}
