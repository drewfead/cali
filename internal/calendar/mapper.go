package calendar

import (
	"time"

	"github.com/drewfead/cali/proto"
	"google.golang.org/api/calendar/v3"
)

// MapProtoToEvent converts a proto AddEventRequest to a Google Calendar Event
func MapProtoToEvent(req *proto.AddEventRequest) *calendar.Event {
	event := &calendar.Event{
		Summary:     req.Title,
		Description: req.Description,
		Location:    req.Location,
	}

	// Determine start time
	var startTime time.Time
	if req.StartTime != nil {
		startTime = req.StartTime.AsTime()
	} else {
		// Default to current time rounded to next hour
		now := time.Now()
		startTime = now.Add(time.Hour - time.Duration(now.Minute())*time.Minute - time.Duration(now.Second())*time.Second)
	}

	// Determine end time
	var endTime time.Time
	if req.EndTime != nil {
		endTime = req.EndTime.AsTime()
	} else {
		// Default to 1 hour after start time
		endTime = startTime.Add(time.Hour)
	}

	// Set event times in RFC3339 format
	event.Start = &calendar.EventDateTime{
		DateTime: startTime.Format(time.RFC3339),
		TimeZone: "UTC",
	}

	event.End = &calendar.EventDateTime{
		DateTime: endTime.Format(time.RFC3339),
		TimeZone: "UTC",
	}

	return event
}
