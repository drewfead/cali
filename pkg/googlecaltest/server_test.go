package googlecaltest

import (
	"context"
	"net/http"
	"testing"
	"time"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func TestMockServer_InsertEvent(t *testing.T) {
	server := NewServer()
	defer server.Close()

	// Create client pointing to mock server
	ctx := context.Background()
	client := &http.Client{}
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(client), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create calendar service: %v", err)
	}

	// Insert an event
	event := &calendar.Event{
		Summary: "Test Event",
		Start: &calendar.EventDateTime{
			DateTime: time.Now().Format(time.RFC3339),
		},
		End: &calendar.EventDateTime{
			DateTime: time.Now().Add(time.Hour).Format(time.RFC3339),
		},
	}

	created, err := svc.Events.Insert("primary", event).Do()
	if err != nil {
		t.Fatalf("failed to insert event: %v", err)
	}

	if created.Id == "" {
		t.Error("expected event ID to be set")
	}
	if created.Summary != "Test Event" {
		t.Errorf("expected summary 'Test Event', got %q", created.Summary)
	}
	if created.Status != "confirmed" {
		t.Errorf("expected status 'confirmed', got %q", created.Status)
	}
}

func TestMockServer_ListEvents(t *testing.T) {
	server := NewServer()
	defer server.Close()

	ctx := context.Background()
	client := &http.Client{}
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(client), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create calendar service: %v", err)
	}

	// Insert multiple events
	baseTime := time.Now()
	for i := 0; i < 5; i++ {
		event := &calendar.Event{
			Summary: "Event " + string(rune('A'+i)),
			Start: &calendar.EventDateTime{
				DateTime: baseTime.Add(time.Duration(i) * time.Hour).Format(time.RFC3339),
			},
			End: &calendar.EventDateTime{
				DateTime: baseTime.Add(time.Duration(i+1) * time.Hour).Format(time.RFC3339),
			},
		}
		_, err := svc.Events.Insert("primary", event).Do()
		if err != nil {
			t.Fatalf("failed to insert event %d: %v", i, err)
		}
	}

	// List all events
	events, err := svc.Events.List("primary").Do()
	if err != nil {
		t.Fatalf("failed to list events: %v", err)
	}

	if len(events.Items) != 5 {
		t.Errorf("expected 5 events, got %d", len(events.Items))
	}
}

func TestMockServer_ListEventsWithPagination(t *testing.T) {
	server := NewServer()
	defer server.Close()

	ctx := context.Background()
	client := &http.Client{}
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(client), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create calendar service: %v", err)
	}

	// Insert 10 events
	baseTime := time.Now()
	for i := 0; i < 10; i++ {
		event := &calendar.Event{
			Summary: "Event " + string(rune('A'+i)),
			Start: &calendar.EventDateTime{
				DateTime: baseTime.Add(time.Duration(i) * time.Hour).Format(time.RFC3339),
			},
			End: &calendar.EventDateTime{
				DateTime: baseTime.Add(time.Duration(i+1) * time.Hour).Format(time.RFC3339),
			},
		}
		_, err := svc.Events.Insert("primary", event).Do()
		if err != nil {
			t.Fatalf("failed to insert event %d: %v", i, err)
		}
	}

	// List with pagination (3 per page)
	var allEvents []*calendar.Event
	pageToken := ""
	for {
		call := svc.Events.List("primary").MaxResults(3)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		events, err := call.Do()
		if err != nil {
			t.Fatalf("failed to list events: %v", err)
		}

		allEvents = append(allEvents, events.Items...)

		if events.NextPageToken == "" {
			break
		}
		pageToken = events.NextPageToken
	}

	if len(allEvents) != 10 {
		t.Errorf("expected 10 total events with pagination, got %d", len(allEvents))
	}
}

func TestMockServer_GetEvent(t *testing.T) {
	server := NewServer()
	defer server.Close()

	ctx := context.Background()
	client := &http.Client{}
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(client), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create calendar service: %v", err)
	}

	// Insert event
	event := &calendar.Event{
		Summary: "Test Event",
		Start: &calendar.EventDateTime{
			DateTime: time.Now().Format(time.RFC3339),
		},
		End: &calendar.EventDateTime{
			DateTime: time.Now().Add(time.Hour).Format(time.RFC3339),
		},
	}

	created, err := svc.Events.Insert("primary", event).Do()
	if err != nil {
		t.Fatalf("failed to insert event: %v", err)
	}

	// Get event by ID
	fetched, err := svc.Events.Get("primary", created.Id).Do()
	if err != nil {
		t.Fatalf("failed to get event: %v", err)
	}

	if fetched.Id != created.Id {
		t.Errorf("expected ID %q, got %q", created.Id, fetched.Id)
	}
	if fetched.Summary != "Test Event" {
		t.Errorf("expected summary 'Test Event', got %q", fetched.Summary)
	}
}

func TestMockServer_DeleteEvent(t *testing.T) {
	server := NewServer()
	defer server.Close()

	ctx := context.Background()
	client := &http.Client{}
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(client), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create calendar service: %v", err)
	}

	// Insert event
	event := &calendar.Event{
		Summary: "Test Event",
		Start: &calendar.EventDateTime{
			DateTime: time.Now().Format(time.RFC3339),
		},
		End: &calendar.EventDateTime{
			DateTime: time.Now().Add(time.Hour).Format(time.RFC3339),
		},
	}

	created, err := svc.Events.Insert("primary", event).Do()
	if err != nil {
		t.Fatalf("failed to insert event: %v", err)
	}

	// Delete event
	err = svc.Events.Delete("primary", created.Id).Do()
	if err != nil {
		t.Fatalf("failed to delete event: %v", err)
	}

	// Verify deletion
	_, err = svc.Events.Get("primary", created.Id).Do()
	if err == nil {
		t.Error("expected error when getting deleted event")
	}
}

func TestMockServer_Reset(t *testing.T) {
	server := NewServer()
	defer server.Close()

	ctx := context.Background()
	client := &http.Client{}
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(client), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create calendar service: %v", err)
	}

	// Insert event
	event := &calendar.Event{
		Summary: "Test Event",
		Start: &calendar.EventDateTime{
			DateTime: time.Now().Format(time.RFC3339),
		},
		End: &calendar.EventDateTime{
			DateTime: time.Now().Add(time.Hour).Format(time.RFC3339),
		},
	}

	_, err = svc.Events.Insert("primary", event).Do()
	if err != nil {
		t.Fatalf("failed to insert event: %v", err)
	}

	// Reset server
	server.Reset()

	// Verify all events are gone
	events, err := svc.Events.List("primary").Do()
	if err != nil {
		t.Fatalf("failed to list events: %v", err)
	}

	if len(events.Items) != 0 {
		t.Errorf("expected 0 events after reset, got %d", len(events.Items))
	}
}
