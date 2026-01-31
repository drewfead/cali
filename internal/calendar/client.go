package calendar

import (
	"context"
	"fmt"
	"net/http"

	"github.com/drewfead/cali/proto"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// Client wraps the Google Calendar API service
type Client struct {
	service *calendar.Service
}

// NewClient creates a new Google Calendar API client.
// Optionally accepts an endpoint URL for testing with mock servers.
func NewClient(ctx context.Context, httpClient *http.Client, endpoint ...string) (*Client, error) {
	opts := []option.ClientOption{option.WithHTTPClient(httpClient)}

	// Add endpoint override if provided
	if len(endpoint) > 0 && endpoint[0] != "" {
		opts = append(opts, option.WithEndpoint(endpoint[0]))
	}

	srv, err := calendar.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to create Calendar service: %w", err)
	}

	return &Client{
		service: srv,
	}, nil
}

// CreateEvent creates a new event in the specified calendar
func (c *Client) CreateEvent(ctx context.Context, req *proto.AddEventRequest) (*calendar.Event, error) {
	// Default to primary calendar if not specified
	calendarID := "primary"
	if req.CalendarId != nil && *req.CalendarId != "" {
		calendarID = *req.CalendarId
	}

	// Convert proto request to Calendar API event
	event := MapProtoToEvent(req)

	// Create the event
	createdEvent, err := c.service.Events.Insert(calendarID, event).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to create event: %w", err)
	}

	return createdEvent, nil
}

// UpdateEvent updates an existing event in the specified calendar
func (c *Client) UpdateEvent(ctx context.Context, req *proto.UpdateEventRequest) (*calendar.Event, error) {
	// Default to primary calendar if not specified
	calendarID := "primary"
	if req.CalendarId != nil && *req.CalendarId != "" {
		calendarID = *req.CalendarId
	}

	// First, get the existing event
	existingEvent, err := c.service.Events.Get(calendarID, req.EventId).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to get event: %w", err)
	}

	// Apply updates from the request
	updatedEvent := MapProtoUpdateToEvent(req, existingEvent)

	// Update the event
	result, err := c.service.Events.Update(calendarID, req.EventId, updatedEvent).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to update event: %w", err)
	}

	return result, nil
}

// DeleteEvent deletes an event from the specified calendar
func (c *Client) DeleteEvent(ctx context.Context, req *proto.DeleteEventRequest) error {
	// Default to primary calendar if not specified
	calendarID := "primary"
	if req.CalendarId != nil && *req.CalendarId != "" {
		calendarID = *req.CalendarId
	}

	// Delete the event
	err := c.service.Events.Delete(calendarID, req.EventId).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("unable to delete event: %w", err)
	}

	return nil
}

// ListEvents returns a channel that streams events from the specified calendar
func (c *Client) ListEvents(ctx context.Context, req *proto.ListEventsRequest) (<-chan *calendar.Event, <-chan error) {
	eventsChan := make(chan *calendar.Event)
	errChan := make(chan error, 1)

	go func() {
		defer close(eventsChan)
		defer close(errChan)

		// Default to primary calendar if not specified
		calendarID := "primary"
		if req.CalendarId != nil && *req.CalendarId != "" {
			calendarID = *req.CalendarId
		}

		// Build the events list call
		call := c.service.Events.List(calendarID).Context(ctx).SingleEvents(true).OrderBy("startTime")

		// Apply time filters if provided
		if req.After != nil && req.After.IsValid() {
			call = call.TimeMin(req.After.AsTime().Format("2006-01-02T15:04:05Z07:00"))
		}
		if req.Before != nil && req.Before.IsValid() {
			call = call.TimeMax(req.Before.AsTime().Format("2006-01-02T15:04:05Z07:00"))
		}

		// Apply limit if specified
		if req.Limit != nil && *req.Limit > 0 {
			call = call.MaxResults(int64(*req.Limit))
		}

		// Paginate through results
		pageToken := ""
		var eventCount int32 = 0

		for {
			if pageToken != "" {
				call = call.PageToken(pageToken)
			}

			events, err := call.Do()
			if err != nil {
				errChan <- fmt.Errorf("unable to retrieve events: %w", err)
				return
			}

			// Stream events to channel
			for _, event := range events.Items {
				// Check if we've hit the limit
				if req.Limit != nil && *req.Limit > 0 && eventCount >= *req.Limit {
					return
				}

				select {
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				case eventsChan <- event:
					eventCount++
				}
			}

			// Check if there are more pages
			pageToken = events.NextPageToken
			if pageToken == "" {
				break
			}
		}
	}()

	return eventsChan, errChan
}
