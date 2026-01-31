package main

import (
	"testing"
	"time"

	"github.com/drewfead/cali/internal/calendar"
	"github.com/drewfead/cali/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestMapProtoToEvent_NewFields(t *testing.T) {
	now := time.Now()
	req := &proto.AddEventRequest{
		Summary:                 "Test Event with All Fields",
		Description:             ptr("<b>HTML Description</b>"),
		Location:                ptr("Conference Room"),
		StartTime:               timestamppb.New(now),
		EndTime:                 timestamppb.New(now.Add(time.Hour)),
		GuestsCanSeeOtherGuests: ptr(true),
		GuestsCanModify:         ptr(true),
		GuestsCanInviteOthers:   ptr(true),
		IdempotencyKey:          ptr("custom-event-id-123"),
		SourceTitle:             ptr("External System"),
		SourceUrl:               ptr("https://example.com/event/123"),
		BlocksTime:              ptr(true),
	}

	event := calendar.MapProtoToEvent(req)

	// Verify basic fields
	if event.Summary != req.Summary {
		t.Errorf("expected title %q, got %q", req.Summary, event.Summary)
	}

	if event.Description != *req.Description {
		t.Errorf("expected description %q, got %q", *req.Description, event.Description)
	}

	if event.Location != *req.Location {
		t.Errorf("expected location %q, got %q", *req.Location, event.Location)
	}

	// Verify idempotency key sets event ID
	if event.Id != *req.IdempotencyKey {
		t.Errorf("expected event ID %q, got %q", *req.IdempotencyKey, event.Id)
	}

	// Verify guest permissions
	if event.GuestsCanSeeOtherGuests == nil || !*event.GuestsCanSeeOtherGuests {
		t.Error("expected GuestsCanSeeOtherGuests to be true")
	}

	if !event.GuestsCanModify {
		t.Error("expected GuestsCanModify to be true")
	}

	if event.GuestsCanInviteOthers == nil || !*event.GuestsCanInviteOthers {
		t.Error("expected GuestsCanInviteOthers to be true")
	}

	// Verify source
	if event.Source == nil {
		t.Fatal("expected Source to be set")
	}
	if event.Source.Title != *req.SourceTitle {
		t.Errorf("expected source title %q, got %q", *req.SourceTitle, event.Source.Title)
	}
	if event.Source.Url != *req.SourceUrl {
		t.Errorf("expected source URL %q, got %q", *req.SourceUrl, event.Source.Url)
	}

	// Verify transparency (blocks time)
	if event.Transparency != "opaque" {
		t.Errorf("expected transparency 'opaque', got %q", event.Transparency)
	}
}

func TestMapProtoToEvent_DefaultTransparency(t *testing.T) {
	req := &proto.AddEventRequest{
		Summary:    "Transparent Event",
		BlocksTime: ptr(false), // Default
	}

	event := calendar.MapProtoToEvent(req)

	if event.Transparency != "transparent" {
		t.Errorf("expected transparency 'transparent', got %q", event.Transparency)
	}
}

func TestMapProtoToEvent_GuestPermissionsDefaults(t *testing.T) {
	req := &proto.AddEventRequest{
		Summary: "Event with Default Permissions",
		// All guest permissions default to false
	}

	event := calendar.MapProtoToEvent(req)

	// When false, these should not be set or should be nil/false
	if event.GuestsCanSeeOtherGuests != nil && *event.GuestsCanSeeOtherGuests {
		t.Error("expected GuestsCanSeeOtherGuests to be false or nil")
	}

	if event.GuestsCanModify {
		t.Error("expected GuestsCanModify to be false")
	}

	if event.GuestsCanInviteOthers != nil && *event.GuestsCanInviteOthers {
		t.Error("expected GuestsCanInviteOthers to be false or nil")
	}
}

func TestMapProtoToEvent_PartialSource(t *testing.T) {
	tests := []struct {
		name        string
		sourceTitle string
		sourceURL   string
		wantSource  bool
	}{
		{
			name:        "only title",
			sourceTitle: "Source Title",
			sourceURL:   "",
			wantSource:  true,
		},
		{
			name:        "only URL",
			sourceTitle: "",
			sourceURL:   "https://example.com",
			wantSource:  true,
		},
		{
			name:        "neither",
			sourceTitle: "",
			sourceURL:   "",
			wantSource:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &proto.AddEventRequest{
				Summary: "Test Event",
			}
			if tt.sourceTitle != "" {
				req.SourceTitle = ptr(tt.sourceTitle)
			}
			if tt.sourceURL != "" {
				req.SourceUrl = ptr(tt.sourceURL)
			}

			event := calendar.MapProtoToEvent(req)

			if tt.wantSource && event.Source == nil {
				t.Error("expected Source to be set")
			}
			if !tt.wantSource && event.Source != nil {
				t.Error("expected Source to be nil")
			}
		})
	}
}
