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

// NewClient creates a new Google Calendar API client
func NewClient(ctx context.Context, httpClient *http.Client) (*Client, error) {
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(httpClient))
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
	calendarID := req.CalendarId
	if calendarID == "" {
		calendarID = "primary"
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
