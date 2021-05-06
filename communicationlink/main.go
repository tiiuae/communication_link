package main

import (
	"context"
	"crypto/tls"
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
	ros "github.com/tiiuae/communication_link/ros"
	types "github.com/tiiuae/communication_link/types"
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

	// attach sigint & sigterm listeners
	terminationSignals := make(chan os.Signal, 1)
	signal.Notify(terminationSignals, syscall.SIGINT, syscall.SIGTERM)

	// quitFunc will be called when process is terminated
	ctx, quitFunc := context.WithCancel(context.Background())

	// wait group will make sure all goroutines have time to clean up
	var wg sync.WaitGroup

	mqttClient := newMQTTClient()
	defer mqttClient.Disconnect(1000)

	localNode := ros.InitRosNode(*deviceID, "communication_link")
	defer localNode.ShutdownRosNode()
	fleetNode := ros.InitRosNode("fleet", "communication_link")
	defer fleetNode.ShutdownRosNode()

	startTelemetry(ctx, &wg, mqttClient, localNode, fleetNode)
	startCommandHandlers(ctx, &wg, mqttClient, localNode)
	publishDefaultMesh(ctx, mqttClient, localNode)

	// wait for termination and close quit to signal all
	<-terminationSignals
	// cancel the main context
	log.Printf("Shuttding down..")
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

func publishDefaultMesh(ctx context.Context, mqttClient mqtt.Client, node *ros.Node) {
	pub := node.InitPublisher("mesh_parameters", "std_msgs/msg/String", (*types.String)(nil))

	text, err := ioutil.ReadFile("./default_mesh.json")
	if err != nil {
		log.Printf("Failed to read default_mesh.json: %v", err)
		return
	}

	pub.DoPublish(types.GenerateString(string(text)))

	pub.Finish()
}
