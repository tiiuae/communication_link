package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	uuid "github.com/google/uuid"
	ros "github.com/tiiuae/communication_link/communicationlink/ros"
	types "github.com/tiiuae/communication_link/communicationlink/types"
)

const (
	qos    = 1
	retain = false
)

type telemetry struct {
	Timestamp int64
	MessageID string

	LocationUpdated  bool
	Lat              float64
	Lon              float64
	Heading          float32
	AltitudeFromHome float32
	DistanceFromHome float32

	BatteryUpdated   bool
	BatteryVoltageV  float32
	BatteryRemaining float32

	StateUpdated bool
	ArmingState  uint8
	NavState     uint8
}

type debugValue struct {
	Updated time.Time
	Value   string
}
type debugValues map[string]debugValue
type sensorData struct {
	SensorData types.SensorCombined
	DeviceID   string
	MessageID  string
}

type batteryData struct {
	Timestamp uint64
	VoltageV  float32
	Remaining float32
	MessageID string
}

type vehicleStatus struct {
	VehicleStatus types.VehicleStatus
	DeviceID      string
	MessageID     string
}

type vehicleLocalPosition struct {
	VehicleLocalPosition types.VehicleLocalPosition
	DeviceID             string
	MessageID            string
}

var rosStartTime uint64 = 0

var (
	telemetryMutex   sync.Mutex
	telemetrySent    bool
	currentTelemetry telemetry
)

// loop to send telemetry 10/s
func startSendingTelemetry(ctx context.Context, mqttClient mqtt.Client) {
	topic := fmt.Sprintf("/devices/%s/%s", *deviceID, "events/telemetry")
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			u := uuid.New()
			telemetryMutex.Lock()
			if telemetrySent {
				// there's no new data to send
				// skip this round
				telemetryMutex.Unlock()
				break
			}
			currentTelemetry.Timestamp = time.Now().UnixNano() / 1000
			currentTelemetry.MessageID = u.String()
			b, _ := json.Marshal(currentTelemetry)
			telemetrySent = true
			currentTelemetry.LocationUpdated = false
			currentTelemetry.StateUpdated = false
			currentTelemetry.BatteryUpdated = false
			telemetryMutex.Unlock()
			mqttClient.Publish(topic, qos, retain, string(b))
		case <-ctx.Done():
			// context cancelled
			return
		}
	}
}

func handleGPSMessages(ctx context.Context, node *ros.Node) {
	messages := make(chan types.VehicleGlobalPosition)
	log.Printf("Creating subscriber for %s", "VehicleGlobalPosition")
	sub := node.InitSubscriber(messages, "VehicleGlobalPosition_PubSubTopic", "px4_msgs/msg/VehicleGlobalPosition")
	go sub.DoSubscribe(ctx)
	for m := range messages {
		telemetryMutex.Lock()
		currentTelemetry.Lat = m.Lat
		currentTelemetry.Lon = m.Lon
		currentTelemetry.LocationUpdated = true
		telemetrySent = false
		telemetryMutex.Unlock()
	}
	sub.Finish()
}

func handleLocalPosMessages(ctx context.Context, node *ros.Node) {
	messages := make(chan types.VehicleLocalPosition)
	log.Printf("Creating subscriber for %s", "VehicleLocalPosition")
	sub := node.InitSubscriber(messages, "VehicleLocalPosition_PubSubTopic", "px4_msgs/msg/VehicleLocalPosition")
	go sub.DoSubscribe(ctx)
	for m := range messages {
		telemetryMutex.Lock()
		currentTelemetry.Heading = m.Heading
		if m.ZValid {
			currentTelemetry.AltitudeFromHome = -m.Z
		} else {
			currentTelemetry.AltitudeFromHome = 0.0
		}
		if m.XyValid {
			currentTelemetry.DistanceFromHome = float32(math.Sqrt(float64(m.X*m.X + m.Y*m.Y)))
		} else {
			currentTelemetry.DistanceFromHome = 0.0
		}
		telemetrySent = false
		telemetryMutex.Unlock()
	}
	sub.Finish()
}

func handleStatusMessages(ctx context.Context, node *ros.Node) {
	messages := make(chan types.VehicleStatus)
	sub := node.InitSubscriber(messages, "VehicleStatus_PubSubTopic", "px4_msgs/msg/VehicleStatus")
	go sub.DoSubscribe(ctx)
	for m := range messages {
		telemetryMutex.Lock()
		currentTelemetry.ArmingState = m.ArmingState
		currentTelemetry.NavState = m.NavState
		currentTelemetry.StateUpdated = true
		telemetrySent = false
		telemetryMutex.Unlock()
	}
	log.Printf("handleStatusMessages END")
	sub.Finish()
}

func handleBatteryMessages(ctx context.Context, node *ros.Node) {
	messages := make(chan types.BatteryStatus)
	log.Printf("Creating subscriber for %s", "BatteryStatus")
	sub := node.InitSubscriber(messages, "BatteryStatus_PubSubTopic", "px4_msgs/msg/BatteryStatus")
	go sub.DoSubscribe(ctx)
	for m := range messages {
		telemetryMutex.Lock()
		currentTelemetry.BatteryVoltageV = m.VoltageV
		currentTelemetry.BatteryRemaining = m.Remaining
		currentTelemetry.BatteryUpdated = true
		telemetrySent = false
		telemetryMutex.Unlock()
	}
}

func handleDebugValues(ctx context.Context, node *ros.Node, mqttClient mqtt.Client) {
	topic := fmt.Sprintf("/devices/%s/%s", *deviceID, "events/debug-values")
	messages := make(chan types.String)
	sub := node.InitSubscriber(messages, "debug_values", "std_msgs/msg/String")
	var currentValues debugValues = make(map[string]debugValue)

	go sub.DoSubscribe(ctx)
	updated := false
	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case m, ok := <-messages:
			if !ok {
				return
			}
			msg := m.GetString()
			parts := strings.SplitN(msg, ":", 3)
			from := parts[0]
			name := parts[1]
			value := parts[2]
			currentValues[from+":"+name] = debugValue{
				Updated: time.Now(),
				Value:   value,
			}
			updated = true
		case <-ticker.C:
			if updated {
				updated = false
				b, _ := json.Marshal(currentValues)
				mqttClient.Publish(topic, qos, retain, string(b))
			}
		}
	}
}

func handleMissionEngine(ctx context.Context, node *ros.Node, mqttClient mqtt.Client) {
	messages := make(chan types.String)
	sub := node.InitSubscriber(messages, "missionengine", "std_msgs/msg/String")
	go sub.DoSubscribe(ctx)
	for m := range messages {
		str := m.GetString()
		var msg missionsMessage
		err := json.Unmarshal([]byte(str), &msg)
		if err != nil {
			log.Printf("Could not unmarshal payload: %v", err)
			continue
		}
		log.Printf("Mission event: %s", msg.MessageType)
		switch msg.MessageType {
		case "mission-plan":
			topic := fmt.Sprintf("/devices/%s/events/mission-plan", msg.From)
			mqttClient.Publish(topic, 1, false, msg.Message)
		case "flight-plan":
			topic := fmt.Sprintf("/devices/%s/events/flight-plan", msg.From)
			mqttClient.Publish(topic, 1, false, msg.Message)
		default:
			log.Printf("Unknown mission event: %s", msg.MessageType)
		}
	}
}

func startTelemetry(ctx context.Context, wg *sync.WaitGroup, mqttClient mqtt.Client, node *ros.Node, fleetNode *ros.Node) {
	wg.Add(7)
	go func() {
		defer wg.Done()
		handleGPSMessages(ctx, node)
	}()
	go func() {
		defer wg.Done()
		handleLocalPosMessages(ctx, node)
	}()
	go func() {
		defer wg.Done()
		handleStatusMessages(ctx, node)
	}()
	go func() {
		defer wg.Done()
		handleBatteryMessages(ctx, node)
	}()

	go func() {
		defer wg.Done()
		startSendingTelemetry(ctx, mqttClient)
	}()
	go func() {
		defer wg.Done()
		handleDebugValues(ctx, node, mqttClient)
	}()
	go func() {
		defer wg.Done()
		handleMissionEngine(ctx, fleetNode, mqttClient)
	}()
}
