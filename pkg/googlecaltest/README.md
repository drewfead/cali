# googlecaltest

A mock Google Calendar API server for testing, built with `httptest`.

## Features

- **No Authentication Required**: Tests run without OAuth or service account credentials
- **Thread-Safe**: Concurrent access is handled with mutexes
- **Full Events API**: Supports Insert, List, Get, Update, Delete operations
- **Pagination**: Implements `maxResults` and `pageToken` query parameters
- **Time Filtering**: Supports `timeMin` and `timeMax` query parameters
- **Sorting**: Handles `orderBy=startTime` with `singleEvents=true`
- **Multiple Calendars**: Each calendar ID maintains separate event storage
- **Test Helpers**: Pre-populate events, get events for assertions, reset state

## Installation

```bash
go get github.com/drewfead/cali/pkg/googlecaltest
```

## Quick Start

```go
import (
    "context"
    "net/http"
    "testing"

    "github.com/drewfead/cali/pkg/googlecaltest"
    "google.golang.org/api/calendar/v3"
    "google.golang.org/api/option"
)

func TestMyCalendarCode(t *testing.T) {
    // Create mock server
    server := googlecaltest.NewServer()
    defer server.Close()

    // Create Google Calendar client pointing to mock
    ctx := context.Background()
    client := &http.Client{}
    svc, err := calendar.NewService(ctx,
        option.WithHTTPClient(client),
        option.WithEndpoint(server.URL))
    if err != nil {
        t.Fatal(err)
    }

    // Use the service normally - no authentication needed!
    event := &calendar.Event{
        Summary: "Test Meeting",
        Start: &calendar.EventDateTime{
            DateTime: time.Now().Format(time.RFC3339),
        },
        End: &calendar.EventDateTime{
            DateTime: time.Now().Add(time.Hour).Format(time.RFC3339),
        },
    }

    created, err := svc.Events.Insert("primary", event).Do()
    if err != nil {
        t.Fatal(err)
    }

    // Assert on created event
    if created.Summary != "Test Meeting" {
        t.Errorf("expected 'Test Meeting', got %q", created.Summary)
    }
}
```

## Supported Operations

### Insert Event
```go
event, err := svc.Events.Insert("primary", &calendar.Event{
    Summary: "New Event",
}).Do()
```

### List Events
```go
// Basic list
events, err := svc.Events.List("primary").Do()

// With pagination
events, err := svc.Events.List("primary").
    MaxResults(10).
    PageToken("token").
    Do()

// With time filtering
events, err := svc.Events.List("primary").
    TimeMin("2024-01-01T00:00:00Z").
    TimeMax("2024-12-31T23:59:59Z").
    Do()

// With sorting
events, err := svc.Events.List("primary").
    SingleEvents(true).
    OrderBy("startTime").
    Do()
```

### Get Event
```go
event, err := svc.Events.Get("primary", "event-id").Do()
```

### Update Event
```go
event, err := svc.Events.Update("primary", "event-id", &calendar.Event{
    Summary: "Updated Event",
}).Do()
```

### Delete Event
```go
err := svc.Events.Delete("primary", "event-id").Do()
```

## Test Helpers

### Pre-populate Events
```go
server.AddEvent("primary", &calendar.Event{
    Id:      "test-event-1",
    Summary: "Existing Event",
    Start: &calendar.EventDateTime{
        DateTime: "2024-01-15T10:00:00Z",
    },
})
```

### Get Events for Assertions
```go
events := server.GetEvents("primary")
if len(events) != 3 {
    t.Errorf("expected 3 events, got %d", len(events))
}
```

### Reset Between Tests
```go
func TestSomething(t *testing.T) {
    server := googlecaltest.NewServer()
    defer server.Close()

    // ... test code ...

    server.Reset() // Clear all data
}
```

## Using with Cali Integration Tests

```go
func TestIntegration_CreateEvent(t *testing.T) {
    // Create mock server
    server := googlecaltest.NewServer()
    defer server.Close()

    // Create calendar client with mock endpoint
    ctx := context.Background()
    httpClient := &http.Client{}
    svc, _ := calendar.NewService(ctx,
        option.WithHTTPClient(httpClient),
        option.WithEndpoint(server.URL))

    calClient, _ := calendar.NewClient(ctx, httpClient)

    // Create your cali service with the mock client
    svc := newCalendarService(cfg)

    // Test without authentication!
    resp, err := svc.AddEvent(ctx, &proto.AddEventRequest{
        Title: "Test Event",
    })

    if err != nil {
        t.Fatal(err)
    }

    // Verify event was created
    events := server.GetEvents("primary")
    if len(events) != 1 {
        t.Errorf("expected 1 event, got %d", len(events))
    }
}
```

## Limitations

- Only implements the Events API (no Calendars, CalendarList, ACL, etc.)
- Simplified pagination (token is just an offset)
- No recurring event expansion
- No timezone handling beyond storing the provided values
- No validation of date/time formats

## Contributing

To add support for more endpoints:

1. Add handler methods to `server.go`
2. Update `handleRequest` routing
3. Add tests to `server_test.go`
4. Update this README

## License

Same as the parent cali project.
