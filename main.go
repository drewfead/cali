package main

import (
	"context"
	"fmt"
	"os"
	"time"

	protocli "github.com/drewfead/proto-cli"
	"github.com/drewfead/cali/proto"
)

type calendarService struct {
	proto.UnimplementedCalendarServiceServer
	events map[string]*proto.AddEventRequest
}

func newCalendarService() *calendarService {
	return &calendarService{
		events: make(map[string]*proto.AddEventRequest),
	}
}

func (s *calendarService) AddEvent(ctx context.Context, req *proto.AddEventRequest) (*proto.AddEventResponse, error) {
	// Generate a simple event ID
	eventID := fmt.Sprintf("event_%d", time.Now().Unix())

	// Store the event
	s.events[eventID] = req

	return &proto.AddEventResponse{
		EventId: eventID,
		Success: true,
		Message: fmt.Sprintf("Event '%s' added successfully", req.Title),
	}, nil
}

func main() {
	ctx := context.Background()
	impl := newCalendarService()

	// Generate CLI from service
	serviceCLI := proto.CalendarServiceServiceCommand(ctx, impl,
		protocli.WithOutputFormats(
			protocli.JSON(),
			protocli.YAML(),
		),
	)

	// Create root command
	rootCmd := protocli.RootCommand("cal",
		protocli.WithService(serviceCLI),
	)

	if err := rootCmd.Run(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
