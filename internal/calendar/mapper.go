package calendar

import (
	"time"

	"github.com/drewfead/cali/proto"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MapProtoToEvent converts a proto AddEventRequest to a Google Calendar Event
func MapProtoToEvent(req *proto.AddEventRequest) *calendar.Event {
	event := &calendar.Event{
		Summary: req.Summary,
	}

	// Set idempotency key as event ID if provided
	if req.IdempotencyKey != nil && *req.IdempotencyKey != "" {
		event.Id = *req.IdempotencyKey
	}

	// Set optional fields if provided
	if req.Description != nil && *req.Description != "" {
		event.Description = *req.Description
	}
	if req.Location != nil && *req.Location != "" {
		event.Location = *req.Location
	}

	// Always explicitly set guest permissions (Google Calendar API defaults differ from our defaults)
	// Google Calendar API uses pointer types for some booleans
	if req.GuestsCanSeeOtherGuests != nil {
		event.GuestsCanSeeOtherGuests = req.GuestsCanSeeOtherGuests
	}
	if req.GuestsCanModify != nil {
		event.GuestsCanModify = *req.GuestsCanModify
	}
	if req.GuestsCanInviteOthers != nil {
		event.GuestsCanInviteOthers = req.GuestsCanInviteOthers
	}

	// Set source if provided
	if (req.SourceTitle != nil && *req.SourceTitle != "") || (req.SourceUrl != nil && *req.SourceUrl != "") {
		event.Source = &calendar.EventSource{}
		if req.SourceTitle != nil {
			event.Source.Title = *req.SourceTitle
		}
		if req.SourceUrl != nil {
			event.Source.Url = *req.SourceUrl
		}
	}

	// Always explicitly set transparency (Google Calendar API defaults may differ)
	// If blocks_time is true, event is "opaque" (blocks time)
	// If blocks_time is false, event is "transparent" (doesn't block time)
	if req.BlocksTime != nil && *req.BlocksTime {
		event.Transparency = "opaque"
	} else {
		event.Transparency = "transparent"
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

// MapProtoUpdateToEvent applies updates from UpdateEventRequest to an existing event
func MapProtoUpdateToEvent(req *proto.UpdateEventRequest, existingEvent *calendar.Event) *calendar.Event {
	// Start with the existing event
	event := existingEvent

	// Update optional fields only if provided
	if req.Summary != nil && *req.Summary != "" {
		event.Summary = *req.Summary
	}
	if req.Description != nil && *req.Description != "" {
		event.Description = *req.Description
	}
	if req.Location != nil && *req.Location != "" {
		event.Location = *req.Location
	}

	// Update guest permissions if provided
	if req.GuestsCanSeeOtherGuests != nil {
		event.GuestsCanSeeOtherGuests = req.GuestsCanSeeOtherGuests
	}
	if req.GuestsCanModify != nil {
		event.GuestsCanModify = *req.GuestsCanModify
	}
	if req.GuestsCanInviteOthers != nil {
		event.GuestsCanInviteOthers = req.GuestsCanInviteOthers
	}

	// Update source if provided
	if req.SourceTitle != nil || req.SourceUrl != nil {
		if event.Source == nil {
			event.Source = &calendar.EventSource{}
		}
		if req.SourceTitle != nil {
			event.Source.Title = *req.SourceTitle
		}
		if req.SourceUrl != nil {
			event.Source.Url = *req.SourceUrl
		}
	}

	// Update transparency if provided
	if req.BlocksTime != nil {
		if *req.BlocksTime {
			event.Transparency = "opaque"
		} else {
			event.Transparency = "transparent"
		}
	}

	// Update start time if provided
	if req.StartTime != nil {
		startTime := req.StartTime.AsTime()
		event.Start = &calendar.EventDateTime{
			DateTime: startTime.Format(time.RFC3339),
			TimeZone: "UTC",
		}
	}

	// Update end time if provided
	if req.EndTime != nil {
		endTime := req.EndTime.AsTime()
		event.End = &calendar.EventDateTime{
			DateTime: endTime.Format(time.RFC3339),
			TimeZone: "UTC",
		}
	}

	return event
}

// MapEventToProto converts a Google Calendar Event to a proto Event
func MapEventToProto(event *calendar.Event, calendarID string) *proto.Event {
	protoEvent := &proto.Event{
		Id:         event.Id,
		Summary:    event.Summary,
		HtmlLink:   event.HtmlLink,
		CalendarId: calendarID,
	}

	// Set optional fields if present
	if event.Description != "" {
		protoEvent.Description = &event.Description
	}
	if event.Location != "" {
		protoEvent.Location = &event.Location
	}
	if event.Status != "" {
		protoEvent.Status = &event.Status
	}
	if event.Transparency != "" {
		protoEvent.Transparency = &event.Transparency
	}

	// Extract organizer information
	if event.Organizer != nil {
		if event.Organizer.Email != "" {
			protoEvent.OrganizerEmail = &event.Organizer.Email
		}
		if event.Organizer.DisplayName != "" {
			protoEvent.OrganizerName = &event.Organizer.DisplayName
		}
	}

	// Extract conference data (primary video link)
	if event.ConferenceData != nil {
		// Get the primary video conference link
		for _, entryPoint := range event.ConferenceData.EntryPoints {
			if entryPoint.EntryPointType == "video" && entryPoint.Uri != "" {
				protoEvent.ConferenceUri = &entryPoint.Uri
				break
			}
		}
		// Get conference ID
		if event.ConferenceData.ConferenceId != "" {
			protoEvent.ConferenceId = &event.ConferenceData.ConferenceId
		}
	}

	// Extract source information
	if event.Source != nil {
		if event.Source.Title != "" {
			protoEvent.SourceTitle = &event.Source.Title
		}
		if event.Source.Url != "" {
			protoEvent.SourceUrl = &event.Source.Url
		}
	}

	// Parse start time
	if event.Start != nil {
		if event.Start.DateTime != "" {
			if t, err := time.Parse(time.RFC3339, event.Start.DateTime); err == nil {
				protoEvent.StartTime = timestamppb.New(t)
			}
		} else if event.Start.Date != "" {
			// All-day event - parse date only
			if t, err := time.Parse("2006-01-02", event.Start.Date); err == nil {
				protoEvent.StartTime = timestamppb.New(t)
			}
		}
	}

	// Parse end time
	if event.End != nil {
		if event.End.DateTime != "" {
			if t, err := time.Parse(time.RFC3339, event.End.DateTime); err == nil {
				protoEvent.EndTime = timestamppb.New(t)
			}
		} else if event.End.Date != "" {
			// All-day event - parse date only
			if t, err := time.Parse("2006-01-02", event.End.Date); err == nil {
				protoEvent.EndTime = timestamppb.New(t)
			}
		}
	}

	// Extract attendee emails
	if event.Attendees != nil {
		for _, attendee := range event.Attendees {
			if attendee.Email != "" {
				protoEvent.Attendees = append(protoEvent.Attendees, attendee.Email)
			}
		}
	}

	return protoEvent
}
