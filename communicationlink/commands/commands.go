package commands

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tiiuae/communication_link/communicationlink/ros2app"
	nav_msgs "github.com/tiiuae/rclgo-msgs/nav_msgs/msg"
	std_msgs "github.com/tiiuae/rclgo-msgs/std_msgs/msg"
	"github.com/tiiuae/rclgo/pkg/rclgo"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"gopkg.in/yaml.v3"
)

const (
	qos    = 1
	retain = false
)

var missionSlug string = ""

var sshID []byte = make([]byte, 0)

type controlCommand struct {
	Command   string
	Payload   string
	Timestamp time.Time
}
type gstreamerCmd struct {
	Command   string
	Address   string
	Timestamp time.Time
}

type trustEvent struct {
	PublicSSHKey string `json:"public_ssh_key"`
}

type missionEvent struct {
	MissionSlug string    `json:"mission_slug"`
	Timestamp   time.Time `json:"timestamp"`
}

type deviceState struct {
	StartedAt time.Time `json:"started_at"`
	Message   string    `json:"message"`
}

type missionsMessage struct {
	Timestamp   time.Time `json:"timestamp"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	ID          string    `json:"id"`
	MessageType string    `json:"message_type"`
	Message     string    `json:"message"`
}

func initializeTrust(client mqtt.Client, deviceID string) {
	//publicKey, privateKey, err := ed25519.GenerateKey(nil)
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	publicKey := &privateKey.PublicKey
	if err != nil {
		log.Print("Could not generate keys")
		return
	}

	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)

	privatePemData := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	})

	// os.Mkdir("ssh", 0755)
	// err = ioutil.WriteFile("ssh/id_rsa", privatePemData, 0600)
	sshID = privatePemData

	sshPublicKey, _ := ssh.NewPublicKey(publicKey)
	sshPublicKeyStr := ssh.MarshalAuthorizedKey(sshPublicKey)

	trust, _ := json.Marshal(trustEvent{
		PublicSSHKey: strings.TrimSuffix(string(sshPublicKeyStr), "\n"),
	})

	// send public key to server
	topic := fmt.Sprintf("/devices/%s/events/trust", deviceID)
	tok := client.Publish(topic, qos, retain, trust)
	if !tok.WaitTimeout(10 * time.Second) {
		log.Printf("Could not send trust within 10s")
		return
	}
	err = tok.Error()
	if err != nil {
		log.Printf("Could not send trust: %v", err)
		return
	}
	log.Printf("Trust initialized")
}

func joinMission(payload []byte, pubMissions *rclgo.Publisher) {
	var info struct {
		GitServerAddress string `json:"git_server_address"`
		GitServerKey     string `json:"git_server_key"`
		MissionSlug      string `json:"mission_slug"`
		SSHID            []byte `json:"ssh_id"`
		SSHKnownHosts    []byte `json:"ssh_known_hosts"`
	}
	err := json.Unmarshal(payload, &info)
	if err != nil {
		log.Printf("Could not unmarshal payload: %v", err)
		return
	}
	log.Printf("Git config: %+v", info)

	// TODO: knownhosts.HashHostname
	knownCloudHost := fmt.Sprintf("%s %s\n", knownhosts.Normalize(info.GitServerAddress), info.GitServerKey)
	// err = ioutil.WriteFile("ssh/known_host_cloud", []byte(knownCloudHost), 0644)
	// if err != nil {
	// 	log.Printf("Could not write known_host_cloud file: %v", err)
	// 	return
	// }

	missionSlug = info.MissionSlug
	info.SSHID = sshID
	info.SSHKnownHosts = []byte(knownCloudHost)

	payload2, err := json.Marshal(info)
	if err != nil {
		log.Printf("Could not marshal payload: %v", err)
		return
	}

	msg := missionsMessage{
		Timestamp:   time.Now().UTC(),
		From:        "self",
		To:          "self",
		ID:          "id1",
		MessageType: "join-mission",
		Message:     string(payload2),
	}
	b, _ := json.Marshal(msg)

	rclErr := pubMissions.Publish(ros2app.CreateString(string(b)))
	if rclErr != nil {
		log.Printf("Failed to publish: %v", rclErr)
	}
}

func leaveMission(payload []byte, pubMissions *rclgo.Publisher) {
	missionSlug = ""
	msg := missionsMessage{
		Timestamp:   time.Now().UTC(),
		From:        "self",
		To:          "self",
		ID:          "id1",
		MessageType: "leave-mission",
		Message:     "",
	}
	b, _ := json.Marshal(msg)

	pubMissions.Publish(ros2app.CreateString(string(b)))
}

func updateBacklog(pubMissions *rclgo.Publisher) {
	msg := missionsMessage{
		Timestamp:   time.Now().UTC(),
		From:        "self",
		To:          "self",
		ID:          "id1",
		MessageType: "update-backlog",
		Message:     "",
	}
	b, _ := json.Marshal(msg)

	pubMissions.Publish(ros2app.CreateString(string(b)))
}

// handleControlCommand takes a command string and forwards it to mavlinkcmd
func handleControlCommand(ctx context.Context, command, deviceID string, mqttClient mqtt.Client, pubMissions *rclgo.Publisher, armingService *rclgo.Client, takeoffService *rclgo.Client, landingService *rclgo.Client) {
	var cmd controlCommand
	err := json.Unmarshal([]byte(command), &cmd)
	if err != nil {
		log.Printf("Could not unmarshal command: %v", err)
		return
	}

	switch cmd.Command {
	case "initialize-trust":
		log.Printf("Initializing trust with backend")
		initializeTrust(mqttClient, deviceID)
	case "join-mission":
		log.Printf("Backend requesting to join a mission")
		joinMission([]byte(cmd.Payload), pubMissions)
	case "leave-mission":
		log.Printf("Backend requesting to leave from mission")
		leaveMission([]byte(cmd.Payload), pubMissions)
	case "update-backlog":
		log.Printf("Backend requesting to update backlog")
		updateBacklog(pubMissions)
	case "takeoff":
		log.Printf("F4F takeoff service call")
		takeoff(ctx, armingService, takeoffService)
	case "land":
		log.Printf("F4F land service call")
		land(ctx, landingService)
	default:
		log.Printf("Unknown command: %v", command)
	}
}

// handleMissionCommand takes a command string and forwards it to mavlinkcmd
func handleMissionCommand(command string, pub *rclgo.Publisher) {
	var cmd controlCommand
	err := json.Unmarshal([]byte(command), &cmd)
	if err != nil {
		log.Printf("Could not unmarshal command: %v", err)
		return
	}
	switch cmd.Command {
	case "new_mission":
		log.Printf("Publishing mission to where ever")
		var path nav_msgs.Path
		pub.Publish(&path)
	default:
		log.Printf("Unknown command: %v", command)
	}
}

// handleGstreamerCommand takes a command string and forwards it to gstreamercmd
func handleGstreamerCommand(command string, pub *rclgo.Publisher) {
	var cmd gstreamerCmd

	err := json.Unmarshal([]byte(command), &cmd)
	if err != nil {
		log.Printf("%v\nCould not unmarshal command: %v", command, err)
		return
	}
	switch cmd.Command {
	case "start":
		log.Printf("Publishing 'start' to /videostreamrcmd")
		pub.Publish(ros2app.CreateString(command))
	case "stop":
		log.Printf("Publishing 'stop' to /videostreamrcmd")
		pub.Publish(ros2app.CreateString(command))
	default:
		log.Printf("Unknown command: %v", command)
	}
}

// handleControlCommands routine waits for commands and executes them. The routine quits when quit channel is closed
func handleControlCommands(ctx context.Context, wg *sync.WaitGroup, mqttClient mqtt.Client, rclContext *rclgo.Context, node *rclgo.Node, commands <-chan string, deviceID string) {
	wg.Add(1)
	defer wg.Done()
	pubMissions := ros2app.NewPublisher(node, "missions", "std_msgs/String")
	armingService := createArmingService(ctx, rclContext, node)
	takeoffService := createTakeoffService(ctx, rclContext, node)
	landingService := createLandingService(ctx, rclContext, node)

	defer pubMissions.Close()
	defer takeoffService.Close()
	defer armingService.Close()

	defer landingService.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case command := <-commands:
			handleControlCommand(ctx, command, deviceID, mqttClient, pubMissions, armingService, takeoffService, landingService)
		}
	}
}

// handleMissionCommands routine waits for commands and executes them. The routine quits when quit channel is closed
func handleMissionCommands(ctx context.Context, wg *sync.WaitGroup, node *rclgo.Node, commands <-chan string) {
	wg.Add(1)
	defer wg.Done()
	pub := ros2app.NewPublisher(node, "whereever", "nav_msgs/Path")
	for {
		select {
		case <-ctx.Done():
			pub.Close()
			return
		case command := <-commands:
			handleMissionCommand(command, pub)
		}
	}
}

// handleGstreamerCommands routine waits for commands and executes them. The routine quits when quit channel is closed
func handleGstreamerCommands(ctx context.Context, wg *sync.WaitGroup, node *rclgo.Node, commands <-chan string) {
	wg.Add(1)
	defer wg.Done()
	pub := ros2app.NewPublisher(node, "videostreamcmd", "std_msgs/String")
	for {
		select {
		case <-ctx.Done():
			pub.Close()
			return
		case command := <-commands:
			handleGstreamerCommand(command, pub)
		}
	}
}

func publishMissionState(ctx context.Context, wg *sync.WaitGroup, mqttClient mqtt.Client, deviceID string) {
	wg.Add(1)
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(15 * time.Second):
			topic := fmt.Sprintf("/devices/%s/events/mission-state", deviceID)
			msg := missionEvent{
				MissionSlug: missionSlug,
				Timestamp:   time.Now().UTC(),
			}
			b, _ := json.Marshal(msg)
			mqttClient.Publish(topic, 1, false, b)
		}
	}
}

func StartCommandHandlers(ctx context.Context, wg *sync.WaitGroup, mqttClient mqtt.Client, rclContext *rclgo.Context, node *rclgo.Node, deviceID string) {
	missionDataRecorderConfigPub := ros2app.NewPublisher(
		node,
		"mission_data_recorder/config",
		"std_msgs/String",
	)

	controlCommands := make(chan string)
	missionCommands := make(chan string)
	gstreamerCommands := make(chan string)

	go handleControlCommands(ctx, wg, mqttClient, rclContext, node, controlCommands, deviceID)
	go handleMissionCommands(ctx, wg, node, missionCommands)
	go handleGstreamerCommands(ctx, wg, node, gstreamerCommands)
	go publishMissionState(ctx, wg, mqttClient, deviceID)

	log.Printf("Subscribing to MQTT commands")
	commandTopic := fmt.Sprintf("/devices/%s/commands/", deviceID)
	token := mqttClient.Subscribe(fmt.Sprintf("%v#", commandTopic), 0, func(client mqtt.Client, msg mqtt.Message) {
		subfolder := strings.TrimPrefix(msg.Topic(), commandTopic)
		switch subfolder {
		case "control":
			log.Printf("Got control command: %v", string(msg.Payload()))
			controlCommands <- string(msg.Payload())
		case "mission":
			log.Printf("Got mission command")
			missionCommands <- string(msg.Payload())
		case "videostream":
			log.Printf("Got videostream command")
			gstreamerCommands <- string(msg.Payload())
		default:
			log.Printf("Unknown command subfolder: %v", subfolder)
		}
	})
	if err := token.Error(); err != nil {
		log.Fatalf("Error on subscribe: %v", err)
	}

	// Latest config received on startup
	configTopic := fmt.Sprintf("/devices/%s/config", deviceID)
	configToken := mqttClient.Subscribe(configTopic, 0, func(client mqtt.Client, msg mqtt.Message) {
		log.Printf("Got config:\n%v", string(msg.Payload()))
		yamlConfig := make(map[string]interface{})
		err := yaml.Unmarshal(msg.Payload(), &yamlConfig)
		if err != nil {
			log.Printf("Failed to unmarshal config yaml: %v", err)
			return
		}
		go publishOTAProfile(node, yamlConfig["profile"])
		go publishMeshConfig(node, yamlConfig["initial-wifi"])
		go publishMissionDataRecorderConfig(
			missionDataRecorderConfigPub,
			yamlConfig["mission-data-recorder"],
		)
	})
	if err := configToken.Error(); err != nil {
		log.Fatalf("Error on subscribe: %v", err)
	}

	publishDeviceState(ctx, mqttClient, deviceID)
}

func publishDeviceState(ctx context.Context, mqttClient mqtt.Client, deviceID string) {
	topic := fmt.Sprintf("/devices/%s/state", deviceID)
	msg := deviceState{
		StartedAt: time.Now().UTC(),
		Message:   "hello world",
	}
	b, _ := json.Marshal(msg)
	mqttClient.Publish(topic, 1, false, b)
}

func publishOTAProfile(node *rclgo.Node, configYaml interface{}) {
	if configYaml == nil {
		log.Println("profile config was not included -> no OTA")
		return
	}

	wifiJson, err := json.Marshal(configYaml)
	if err != nil {
		log.Printf("Failed to marshal profile json: %v", err)
		return
	}

	log.Printf("Publishing OTA profile")
	opts := rclgo.NewDefaultPublisherOptions()
	opts.Qos.Reliability = rclgo.RmwQosReliabilityPolicySystemDefault
	pub, _ := node.NewPublisher("ota_profile", std_msgs.StringTypeSupport, opts)
	time.Sleep(5 * time.Second)
	err = pub.Publish(ros2app.CreateString(string(wifiJson)))
	if err != nil {
		log.Printf("Failed to publish OTA profile: %v", err)
		return
	}
	log.Printf("OTA profile published")
}

func publishMeshConfig(node *rclgo.Node, configYaml interface{}) {
	if configYaml == nil {
		log.Println("initial-wifi config was not included -> mesh not configured")
		return
	}

	wifiJson, err := json.Marshal(configYaml)
	if err != nil {
		log.Printf("Failed to marshal mesh json: %v", err)
		return
	}

	log.Printf("Publishing mesh parameters")
	opts := rclgo.NewDefaultPublisherOptions()
	opts.Qos.Durability = rclgo.RmwQosDurabilityPolicyTransientLocal
	opts.Qos.Reliability = rclgo.RmwQosReliabilityPolicyReliable
	pub, _ := node.NewPublisher("mesh_parameters", std_msgs.StringTypeSupport, opts)
	time.Sleep(5 * time.Second)
	err = pub.Publish(ros2app.CreateString(string(wifiJson)))
	if err != nil {
		log.Printf("Failed to publish mesh json: %v", err)
		return
	}
	log.Printf("Mesh parameters published")
}

func publishMissionDataRecorderConfig(pub *rclgo.Publisher, configYaml interface{}) {
	if configYaml == nil {
		log.Println("mission-data-recorder config was not included")
		return
	}
	config, err := yaml.Marshal(configYaml)
	if err != nil {
		log.Println("Failed to encode mission-data-recorder config:", err)
		return
	}
	if err = pub.Publish(ros2app.CreateString(string(config))); err != nil {
		log.Println("Failed to publish mission-data-recorder config:", err)
		return
	}
	log.Println("mission-data-recorder config published")
}
