package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/tiiuae/communication_link/missionengine/internal/commands"
	"github.com/tiiuae/communication_link/missionengine/internal/fleet"
	"github.com/tiiuae/communication_link/missionengine/internal/flyf4f"
	"github.com/tiiuae/communication_link/missionengine/internal/gittransport"
	"github.com/tiiuae/communication_link/missionengine/internal/missionplanner"
	"github.com/tiiuae/communication_link/missionengine/internal/telemetry"
	"github.com/tiiuae/communication_link/missionengine/internal/types"

	"github.com/tiiuae/rclgo/pkg/ros2"
)

var (
	deafultFlagSet    = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	deviceID          = deafultFlagSet.String("device_id", "", "The provisioned device id")
	mqttBrokerAddress = deafultFlagSet.String("mqtt_broker", "", "MQTT broker protocol, address and port")
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

	rclLocalNode, rclErr := rclContext.NewNode("missionengine_local", *deviceID)
	if rclErr != nil {
		log.Fatal(rclErr)
	}

	rclFleetNode, rclErr := rclContext.NewNode("missionengine_fleet", "fleet")
	if rclErr != nil {
		log.Fatal(rclErr)
	}

	messagebus := make(chan types.Message, 100)
	bus := types.NewMessageBus(
		messagebus,
		types.NewLogger(),
		telemetry.New(rclLocalNode, *deviceID),
		commands.New(rclLocalNode, *deviceID),
		gittransport.New(*deviceID),
		missionplanner.New(*deviceID),
		fleet.New(rclFleetNode, *deviceID),
		// flypx4.New(rclLocalNode, *deviceID),
		flyf4f.New(ctx, rclContext, rclLocalNode, *deviceID),
	)

	go bus.Run(ctx, &wg)

	// wait for termination and close quit to signal all
	<-terminationSignals
	// cancel the main context
	log.Printf("Shutting down..")
	quitFunc()

	// wait until goroutines have done their cleanup
	log.Printf("Waiting for routines to finish...")
	wg.Wait()
	log.Printf("Signing off - BYE")
}
