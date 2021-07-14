package commands

import (
	"context"
	"log"
	"time"

	std_srvs "github.com/tiiuae/rclgo-msgs/std_srvs/srv"
	"github.com/tiiuae/rclgo/pkg/rclgo"
)

func takeoff(ctx context.Context, armingService *rclgo.Client, takeoffService *rclgo.Client) {
	arm(ctx, armingService)
	req := std_srvs.NewTrigger_Request()
	res, _, err := takeoffService.Send(ctx, req)
	if err != nil {
		log.Printf("%v", err)
	}
	log.Printf("F4F: TAKEOFF: %v", res)
}

func arm(ctx context.Context, armingService *rclgo.Client) {
	req := std_srvs.NewSetBool_Request()
	req.Data = true
	res, _, err := armingService.Send(ctx, req)
	if err != nil {
		log.Fatalf("%v", err)
	}
	log.Printf("F4F: ARMING: %v", res)
}

func land(ctx context.Context, landingService *rclgo.Client) {
	req := std_srvs.NewTrigger_Request()
	res, _, err := landingService.Send(ctx, req)
	if err != nil {
		log.Printf("%v", err)
	}
	log.Printf("F4F: LAND: %v", res)
}

func createTakeoffService(ctx context.Context, rclContext *rclgo.Context, node *rclgo.Node) *rclgo.Client {
	opt := &rclgo.ClientOptions{Qos: rclgo.NewRmwQosProfileServicesDefault()}
	client, err := node.NewClient("control_interface/takeoff", std_srvs.TriggerTypeSupport, opt)
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

func createLandingService(ctx context.Context, rclContext *rclgo.Context, node *rclgo.Node) *rclgo.Client {
	opt := &rclgo.ClientOptions{Qos: rclgo.NewRmwQosProfileServicesDefault()}
	client, err := node.NewClient("control_interface/land", std_srvs.TriggerTypeSupport, opt)
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

func createArmingService(ctx context.Context, rclContext *rclgo.Context, node *rclgo.Node) *rclgo.Client {
	opt := &rclgo.ClientOptions{Qos: rclgo.NewRmwQosProfileServicesDefault()}
	client, err := node.NewClient("control_interface/arming", std_srvs.SetBoolTypeSupport, opt)
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
