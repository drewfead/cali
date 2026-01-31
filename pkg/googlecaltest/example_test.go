package googlecaltest_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/drewfead/cali/pkg/googlecaltest"
	"github.com/drewfead/cali/proto"
	"google.golang.org/api/option"
	gcalendar "google.golang.org/api/calendar/v3"
)

// ptr returns a pointer to the given value (helper for proto optional fields)
func ptr[T any](v T) *T {
	return &v
}

// Example demonstrates how to use the mock server with cali's calendar client.
func Example() {
	// Create mock server
	server := googlecaltest.NewServer()
	defer server.Close()

	// Create Google Calendar service pointing to mock
	ctx := context.Background()
	httpClient := &http.Client{}
	svc, err := gcalendar.NewService(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(server.URL))
	if err != nil {
		panic(err)
	}

	// Pre-populate some events
	server.AddEvent("primary", &gcalendar.Event{
		Id:      "event1",
		Summary: "Team Meeting",
		Start: &gcalendar.EventDateTime{
			DateTime: time.Now().Format(time.RFC3339),
		},
		End: &gcalendar.EventDateTime{
			DateTime: time.Now().Add(time.Hour).Format(time.RFC3339),
		},
	})

	// Use the service
	events, err := svc.Events.List("primary").Do()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Found %d events\n", len(events.Items))
	// Output: Found 1 events
}

// Example_protoRequest shows how to test the full request flow.
func Example_protoRequest() {
	server := googlecaltest.NewServer()
	defer server.Close()

	ctx := context.Background()
	httpClient := &http.Client{}
	svc, err := gcalendar.NewService(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(server.URL))
	if err != nil {
		panic(err)
	}

	// Simulate a proto AddEventRequest
	req := &proto.AddEventRequest{
		Summary:     "Test Event",
		Description: ptr("Integration test event"),
		Location:    ptr("Conference Room A"),
	}

	// Convert proto to calendar event (using mapper logic)
	event := &gcalendar.Event{
		Summary:     req.Summary,
		Description: *req.Description,
		Location:    *req.Location,
		Start: &gcalendar.EventDateTime{
			DateTime: time.Now().Format(time.RFC3339),
		},
		End: &gcalendar.EventDateTime{
			DateTime: time.Now().Add(time.Hour).Format(time.RFC3339),
		},
	}

	// Insert event
	created, err := svc.Events.Insert("primary", event).Do()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Event created: %s\n", created.Id)
	fmt.Printf("Summary: %s\n", created.Summary)
	// Output:
	// Event created: event1
	// Summary: Test Event
}
