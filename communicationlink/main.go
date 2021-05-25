package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tiiuae/communication_link/communicationlink/commands"
	"github.com/tiiuae/communication_link/communicationlink/ros2app"
	"github.com/tiiuae/communication_link/communicationlink/telemetry"
	"github.com/tiiuae/rclgo/pkg/ros2"
)

const (
	registryID    = "fleet-registry"
	projectID     = "auto-fleet-mgnt"
	region        = "europe-west1"
	algorithm     = "RS256"
	defaultServer = "ssl://mqtt.googleapis.com:8883"
)

var (
	deafultFlagSet    = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	deviceID          = deafultFlagSet.String("device_id", "", "The provisioned device id")
	mqttBrokerAddress = deafultFlagSet.String("mqtt_broker", "", "MQTT broker protocol, address and port")
	privateKeyPath    = deafultFlagSet.String("private_key", "/enclave/rsa_private.pem", "The private key for the MQTT authentication")
)

// MQTT parameters
const (
	TopicType = "events" // or "state"
	QoS       = 1        // QoS 2 isn't supported in GCP
	Retain    = false
	Username  = "unused" // always this value in GCP
)

func main() {
	deafultFlagSet.Parse(os.Args[1:])

	terminationSignals := make(chan os.Signal, 1)
	signal.Notify(terminationSignals, syscall.SIGINT, syscall.SIGTERM)
	ctx, quitFunc := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// Setup MQTT
	mqttClient := newMQTTClient()
	defer mqttClient.Disconnect(1000)

	// Setup ROS nodes
	rclArgs, rclErr := ros2.NewRCLArgs("")
	if rclErr != nil {
		log.Fatal(rclErr)
	}

	rclContext, rclErr := ros2.NewContext(&wg, 0, rclArgs)
	if rclErr != nil {
		log.Fatal(rclErr)
	}
	defer rclContext.Close()

	rclLocalNode, rclErr := rclContext.NewNode("communicationlink_local", *deviceID)
	if rclErr != nil {
		log.Fatal(rclErr)
	}

	rclFleetNode, rclErr := rclContext.NewNode("communicationlink_fleet", "fleet")
	if rclErr != nil {
		log.Fatal(rclErr)
	}

	localSubs := ros2app.NewSubscriptions(rclLocalNode)
	fleetSubs := ros2app.NewSubscriptions(rclFleetNode)

	// Setup telemetry
	telemetry.RegisterLocalSubscriptions(localSubs, ctx, mqttClient, *deviceID)
	telemetry.RegisterFleetSubscriptions(fleetSubs, ctx, mqttClient, *deviceID)

	err := localSubs.Subscribe(ctx)
	if err != nil {
		log.Fatal(err)
	}

	err = fleetSubs.Subscribe(ctx)
	if err != nil {
		log.Fatal(err)
	}

	telemetry.Start(ctx, &wg, mqttClient, *deviceID)

	// Setup commandhandlers
	commands.StartCommandHandlers(ctx, &wg, mqttClient, rclLocalNode, *deviceID)

	// Setup mesh
	// publishDefaultMesh(ctx, mqttClient, rclLocalNode, *deviceID)

	// wait for termination and close quit to signal all
	<-terminationSignals
	// cancel the main context
	log.Printf("Shutting down..")
	quitFunc()
	// wait until goroutines have done their cleanup
	log.Printf("Waiting for routines to finish..")
	wg.Wait()
	log.Printf("Signing off - BYE")
}

func newMQTTClient() mqtt.Client {
	serverAddress := *mqttBrokerAddress
	if serverAddress == "" {
		serverAddress = defaultServer
	}
	log.Printf("address: %v", serverAddress)

	// generate MQTT client
	clientID := fmt.Sprintf(
		"projects/%s/locations/%s/registries/%s/devices/%s",
		projectID, region, registryID, *deviceID)

	log.Println("Client ID:", clientID)

	// load private key
	keyData, err := ioutil.ReadFile(*privateKeyPath)
	if err != nil {
		panic(err)
	}

	var key interface{}
	switch algorithm {
	case "RS256":
		key, err = jwt.ParseRSAPrivateKeyFromPEM(keyData)
	case "ES256":
		key, err = jwt.ParseECPrivateKeyFromPEM(keyData)
	default:
		log.Fatalf("Unknown algorithm: %s", algorithm)
	}
	if err != nil {
		panic(err)
	}

	// generate JWT as the MQTT password
	t := time.Now()
	token := jwt.NewWithClaims(jwt.GetSigningMethod(algorithm), &jwt.StandardClaims{
		IssuedAt:  t.Unix(),
		ExpiresAt: t.Add(24 * time.Hour).Unix(),
		Audience:  projectID,
	})
	pass, err := token.SignedString(key)
	if err != nil {
		panic(err)
	}

	// configure MQTT client
	opts := mqtt.NewClientOptions().
		AddBroker(serverAddress).
		SetClientID(clientID).
		SetUsername(Username).
		SetTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12}).
		SetPassword(pass).
		SetProtocolVersion(4) // Use MQTT 3.1.1

	client := mqtt.NewClient(opts)

	for {
		// retury for ever
		// connect to GCP Cloud IoT Core
		log.Printf("Connecting MQTT...")
		tok := client.Connect()
		if err := tok.Error(); err != nil {
			panic(err)
		}
		if !tok.WaitTimeout(time.Second * 5) {
			log.Println("Connection Timeout")
			continue
		}
		if err := tok.Error(); err != nil {
			panic(err)
		}
		log.Printf("..Connected")
		break
	}

	// need mqtt reconnect each 120 minutes for long use

	return client
}

func publishDefaultMesh(ctx context.Context, mqttClient mqtt.Client, node *ros2.Node, deviceID string) {
	go func() {
		log.Printf("Sending mesh parameters")
		pub := ros2app.NewPublisher(node, "mesh_parameters", "std_msgs/String")

		text, err := ioutil.ReadFile("./default_mesh.json")
		if err != nil {
			pub.Publish(ros2app.CreateString(string(defaultMesh(deviceID))))
			log.Printf("Mesh parameters sent")
			return
		}

		pub.Publish(ros2app.CreateString(string(text)))
		log.Printf("Mesh parameters sent")
	}()
}

func defaultMesh(deviceID string) string {
	mesh := map[string]interface{}{
		"api_version": 1,
		"ssid":        "gold",
		"key":         "1234567890",
		"enc":         "wep",
		"ap_mac":      "00:11:22:33:44:55",
		"country":     "fi",
		"frequency":   "5220",
		"ip":          fmt.Sprintf("192.168.1.%s", string(deviceID[len(deviceID)-1])),
		"subnet":      "255.255.255.0",
		"tx_power":    "30",
		"mode":        "mesh",
	}

	res, _ := json.Marshal(mesh)

	return string(res)
}
