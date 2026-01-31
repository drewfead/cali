// Package googlecaltest provides a mock Google Calendar API server for testing.
//
// The mock server implements a subset of the Google Calendar API v3 Events endpoints,
// allowing tests to run without authentication or network access.
//
// # Supported Operations
//
// The mock server supports the following Google Calendar API operations:
//
//   - Insert Event: POST /calendars/{calendarId}/events
//   - List Events: GET /calendars/{calendarId}/events (with pagination, time filters, sorting)
//   - Get Event: GET /calendars/{calendarId}/events/{eventId}
//   - Update Event: PUT/PATCH /calendars/{calendarId}/events/{eventId}
//   - Delete Event: DELETE /calendars/{calendarId}/events/{eventId}
//
// # Basic Usage
//
//	// Create mock server
//	server := googlecaltest.NewServer()
//	defer server.Close()
//
//	// Create Google Calendar client pointing to mock
//	ctx := context.Background()
//	client := &http.Client{}
//	svc, err := calendar.NewService(ctx,
//	    option.WithHTTPClient(client),
//	    option.WithEndpoint(server.URL))
//
//	// Use the service normally
//	event := &calendar.Event{
//	    Summary: "Test Meeting",
//	    Start: &calendar.EventDateTime{
//	        DateTime: time.Now().Format(time.RFC3339),
//	    },
//	}
//	created, err := svc.Events.Insert("primary", event).Do()
//
// # Test Helpers
//
// The server provides helper methods for test setup and assertions:
//
//	// Pre-populate events for testing
//	server.AddEvent("primary", &calendar.Event{
//	    Id: "test-event-1",
//	    Summary: "Existing Event",
//	})
//
//	// Get all events for assertions
//	events := server.GetEvents("primary")
//
//	// Clear all data between tests
//	server.Reset()
//
// # Features
//
//   - Thread-safe: Uses mutex for concurrent access
//   - Pagination: Supports maxResults and pageToken query parameters
//   - Time filtering: Supports timeMin and timeMax query parameters
//   - Sorting: Supports orderBy=startTime with singleEvents=true
//   - Multiple calendars: Each calendar ID maintains separate event storage
//   - Automatic ID generation: Assigns sequential IDs to new events
//   - Metadata: Sets Created, Updated, Status, and HtmlLink fields
package googlecaltest
