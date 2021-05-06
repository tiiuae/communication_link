package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	ros "github.com/tiiuae/communication_link/missionengine/ros"
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

	localNode := ros.InitRosNode(*deviceID, "mission_engine")
	defer localNode.ShutdownRosNode()
	fleetNode := ros.InitRosNode("fleet", "mission_engine")
	defer fleetNode.ShutdownRosNode()

	me := New(ctx, &wg, localNode, fleetNode, *deviceID)
	startCommandHandlers(ctx, &wg, me, localNode)

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
