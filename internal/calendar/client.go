package calendar

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

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

// GetEvent retrieves a single event by ID
func (c *Client) GetEvent(ctx context.Context, req *proto.GetEventRequest) (*calendar.Event, error) {
	// Default to primary calendar if not specified
	calendarID := "primary"
	if req.CalendarId != nil && *req.CalendarId != "" {
		calendarID = *req.CalendarId
	}

	event, err := c.service.Events.Get(calendarID, req.EventId).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to get event: %w", err)
	}
	return event, nil
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

// ListEvents returns a channel that streams events from the specified calendar with pagination support
func (c *Client) ListEvents(ctx context.Context, req *proto.ListEventsRequest) (<-chan *proto.ListEventsResponse, <-chan error) {
	responseChan := make(chan *proto.ListEventsResponse)
	errChan := make(chan error, 1)

	go func() {
		defer close(responseChan)
		defer close(errChan)

		// Default to primary calendar if not specified
		calendarID := "primary"
		if req.CalendarId != nil && *req.CalendarId != "" {
			calendarID = *req.CalendarId
		}

		slog.Debug("listing events", "calendar_id", calendarID)

		// Build the events list call
		call := c.service.Events.List(calendarID).Context(ctx).SingleEvents(true)

		// Apply time filters based on flags
		// Priority: explicit after/before > boolean flags (future/past) > default (all events)
		// Note: Check for non-zero timestamps, not just IsValid(), since protobuf creates zero-value timestamps
		hasExplicitTimes := (req.After != nil && req.After.IsValid() && req.After.AsTime().Unix() > 0) ||
			(req.Before != nil && req.Before.IsValid() && req.Before.AsTime().Unix() > 0)
		hasTimeFilter := false

		if hasExplicitTimes {
			// Use explicit after/before timestamps
			if req.After != nil && req.After.IsValid() && req.After.AsTime().Unix() > 0 {
				call = call.TimeMin(req.After.AsTime().Format("2006-01-02T15:04:05Z07:00"))
				hasTimeFilter = true
			}
			if req.Before != nil && req.Before.IsValid() && req.Before.AsTime().Unix() > 0 {
				call = call.TimeMax(req.Before.AsTime().Format("2006-01-02T15:04:05Z07:00"))
				hasTimeFilter = true
			}
		} else if req.Future != nil && *req.Future {
			// Future events (after now)
			call = call.TimeMin(time.Now().Format("2006-01-02T15:04:05Z07:00"))
			hasTimeFilter = true
		} else if req.Past != nil && *req.Past {
			// Past events (before now)
			call = call.TimeMax(time.Now().Format("2006-01-02T15:04:05Z07:00"))
			hasTimeFilter = true
		}
		// else: no time filter (all events)

		// Only use orderBy when we have a time filter (required by Google Calendar API)
		if hasTimeFilter {
			call = call.OrderBy("startTime")
		}

		// Apply limit if specified (page size)
		if req.Limit != nil && *req.Limit > 0 {
			call = call.MaxResults(int64(*req.Limit))
		}

		// Use provided anchor if specified
		if req.Anchor != nil && *req.Anchor != "" {
			call = call.PageToken(*req.Anchor)
		}

		// Fetch one page of results
		events, err := call.Do()
		if err != nil {
			slog.Error("failed to retrieve events", "error", err, "calendar_id", calendarID)
			errChan <- fmt.Errorf("unable to retrieve events: %w", err)
			return
		}

		slog.Debug("retrieved events", "count", len(events.Items), "has_next_page", events.NextPageToken != "")

		// Stream events to channel
		for _, event := range events.Items {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			case responseChan <- &proto.ListEventsResponse{
				Event: MapEventToProto(event, calendarID),
			}:
			}
		}

		// Send final message with next_anchor if there are more results
		if events.NextPageToken != "" {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			case responseChan <- &proto.ListEventsResponse{
				NextAnchor: &events.NextPageToken,
			}:
			}
		}
	}()

	return responseChan, errChan
}
