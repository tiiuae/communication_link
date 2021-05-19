package telemetry

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

	"github.com/tiiuae/communication_link/communicationlink/ros2app"

	"github.com/tiiuae/rclgo/pkg/ros2"
	_ "github.com/tiiuae/rclgo/pkg/ros2/msgs"
	px4_msgs "github.com/tiiuae/rclgo/pkg/ros2/msgs/px4_msgs/msg"
	std_msgs "github.com/tiiuae/rclgo/pkg/ros2/msgs/std_msgs/msg"
)

func RegisterLocalSubscriptions(subs *ros2app.Subscriptions, ctx context.Context, mqttClient mqtt.Client, deviceID string) {
	subs.Add("VehicleGlobalPosition_PubSubTopic", "px4_msgs/VehicleGlobalPosition", handleGPSMessages)
	subs.Add("VehicleLocalPosition_PubSubTopic", "px4_msgs/VehicleLocalPosition", handleLocalPosMessages)
	subs.Add("VehicleStatus_PubSubTopic", "px4_msgs/VehicleStatus", handleStatusMessages)
	subs.Add("BatteryStatus_PubSubTopic", "px4_msgs/BatteryStatus", handleBatteryMessages)
	subs.Add("debug_values", "std_msgs/String", handleDebugValues(ctx, mqttClient, deviceID))
}

func RegisterFleetSubscriptions(subs *ros2app.Subscriptions, ctx context.Context, mqttClient mqtt.Client, deviceID string) {
	subs.Add("missionengine", "std_msgs/String", handleMissionEngineMessages(ctx, mqttClient, deviceID))
}

func Start(ctx context.Context, wg *sync.WaitGroup, mqttClient mqtt.Client, deviceID string) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		startSendingTelemetry(ctx, mqttClient, deviceID)
	}()
}

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
	SensorData px4_msgs.SensorCombined
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
	VehicleStatus px4_msgs.VehicleStatus
	DeviceID      string
	MessageID     string
}

type vehicleLocalPosition struct {
	VehicleLocalPosition px4_msgs.VehicleLocalPosition
	DeviceID             string
	MessageID            string
}

type missionsMessage struct {
	Timestamp   time.Time `json:"timestamp"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	ID          string    `json:"id"`
	MessageType string    `json:"message_type"`
	Message     string    `json:"message"`
}

var rosStartTime uint64 = 0

var (
	telemetryMutex   sync.Mutex
	telemetrySent    bool
	currentTelemetry telemetry
)

// loop to send telemetry 10/s
func startSendingTelemetry(ctx context.Context, mqttClient mqtt.Client, deviceID string) {
	topic := fmt.Sprintf("/devices/%s/%s", deviceID, "events/telemetry")
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
			return
		}
	}
}

func handleGPSMessages(s *ros2.Subscription) {
	var m px4_msgs.VehicleGlobalPosition
	_, err := s.TakeMessage(&m)
	if err != nil {
		log.Print("TakeMessage failed: handleGPSMessages")
		return
	}

	telemetryMutex.Lock()
	currentTelemetry.Lat = m.Lat
	currentTelemetry.Lon = m.Lon
	currentTelemetry.LocationUpdated = true
	telemetrySent = false
	telemetryMutex.Unlock()
}

func handleLocalPosMessages(s *ros2.Subscription) {
	var m px4_msgs.VehicleLocalPosition
	_, err := s.TakeMessage(&m)
	if err != nil {
		log.Print("TakeMessage failed: handleLocalPosMessages")
		return
	}

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

func handleStatusMessages(s *ros2.Subscription) {
	var m px4_msgs.VehicleStatus
	_, err := s.TakeMessage(&m)
	if err != nil {
		log.Print("TakeMessage failed: handleStatusMessages")
		return
	}

	telemetryMutex.Lock()
	currentTelemetry.ArmingState = m.ArmingState
	currentTelemetry.NavState = m.NavState
	currentTelemetry.StateUpdated = true
	telemetrySent = false
	telemetryMutex.Unlock()
}

func handleBatteryMessages(s *ros2.Subscription) {
	var m px4_msgs.BatteryStatus
	_, err := s.TakeMessage(&m)
	if err != nil {
		log.Print("TakeMessage failed: handleBatteryMessages")
		return
	}

	telemetryMutex.Lock()
	currentTelemetry.BatteryVoltageV = m.VoltageV
	currentTelemetry.BatteryRemaining = m.Remaining
	currentTelemetry.BatteryUpdated = true
	telemetrySent = false
	telemetryMutex.Unlock()
}

func handleDebugValues(ctx context.Context, mqttClient mqtt.Client, deviceID string) func(s *ros2.Subscription) {
	topic := fmt.Sprintf("/devices/%s/%s", deviceID, "events/debug-values")
	var mutex sync.Mutex
	var currentValues debugValues = make(map[string]debugValue)
	updated := false
	ticker := time.NewTicker(100 * time.Millisecond)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if updated {
					updated = false
					b, _ := json.Marshal(currentValues)
					mqttClient.Publish(topic, qos, retain, string(b))
				}
			}
		}
	}()
	return func(s *ros2.Subscription) {
		var m std_msgs.String
		_, err := s.TakeMessage(&m)
		if err != nil {
			log.Print("TakeMessage failed: handleDebugValues")
			return
		}

		mutex.Lock()
		msg := fmt.Sprintf("%v", m.Data)
		parts := strings.SplitN(msg, ":", 3)
		from := parts[0]
		name := parts[1]
		value := parts[2]
		currentValues[from+":"+name] = debugValue{
			Updated: time.Now(),
			Value:   value,
		}
		updated = true
		mutex.Unlock()
	}
}

func handleMissionEngineMessages(ctx context.Context, mqttClient mqtt.Client, deviceID string) func(s *ros2.Subscription) {
	return func(s *ros2.Subscription) {
		var m std_msgs.String
		_, rclErr := s.TakeMessage(&m)
		if rclErr != nil {
			log.Print("TakeMessage failed: handleMissionEngineMessages")
			return
		}

		str := fmt.Sprintf("%v", m.Data)

		var msg missionsMessage
		err := json.Unmarshal([]byte(str), &msg)
		if err != nil {
			log.Printf("Could not unmarshal payload: %v", err)
			return
		}

		// Mission events are coming from entire fleet. Publish only local messages to cloud.
		if msg.From != deviceID {
			return
		}

		log.Printf("Mission event: %s", msg.MessageType)
		switch msg.MessageType {
		case "mission-plan":
			topic := fmt.Sprintf("/devices/%s/events/mission-plan", msg.From)
			mqttClient.Publish(topic, 1, false, msg.Message)
		case "flight-plan":
			topic := fmt.Sprintf("/devices/%s/events/flight-plan", msg.From)
			mqttClient.Publish(topic, 1, false, msg.Message)
		case "joined-mission":
			topic := fmt.Sprintf("/devices/%s/events/joined-mission", msg.From)
			mqttClient.Publish(topic, 1, false, msg.Message)
		case "left-mission":
			topic := fmt.Sprintf("/devices/%s/events/left-mission", msg.From)
			mqttClient.Publish(topic, 1, false, msg.Message)
		default:
			log.Printf("Unknown mission event: %s", msg.MessageType)
		}
	}
}
