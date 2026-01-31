// Package googlecaltest provides a mock Google Calendar API server for testing.
// It implements a subset of the Google Calendar API v3 Events endpoints.
package googlecaltest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"time"

	"google.golang.org/api/calendar/v3"
)

// Server is a mock Google Calendar API server for testing.
type Server struct {
	*httptest.Server
	mu       sync.RWMutex
	events   map[string]map[string]*calendar.Event // calendarID -> eventID -> event
	nextID   int
	baseTime time.Time
}

// NewServer creates a new mock Google Calendar API server.
func NewServer() *Server {
	s := &Server{
		events:   make(map[string]map[string]*calendar.Event),
		nextID:   1,
		baseTime: time.Now(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRequest)

	s.Server = httptest.NewServer(mux)
	return s
}

// handleRequest routes all requests.
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Check if this is a calendar events request
	if !strings.Contains(r.URL.Path, "/calendars/") || !strings.Contains(r.URL.Path, "/events") {
		http.Error(w, "unsupported endpoint", http.StatusNotFound)
		return
	}
	s.handleCalendars(w, r)
}

// handleCalendars routes calendar-related requests.
func (s *Server) handleCalendars(w http.ResponseWriter, r *http.Request) {
	// Parse URL: /calendar/v3/calendars/{calendarId}/events[/{eventId}]
	path := r.URL.Path

	// Find the calendars section
	idx := strings.Index(path, "/calendars/")
	if idx == -1 {
		http.Error(w, "invalid path: missing /calendars/", http.StatusBadRequest)
		return
	}

	// Extract everything after /calendars/
	path = path[idx+len("/calendars/"):]
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) < 2 {
		http.Error(w, fmt.Sprintf("invalid path: expected at least calendarId/resource, got %v", parts), http.StatusBadRequest)
		return
	}

	calendarID := parts[0]
	resource := parts[1]

	if resource != "events" {
		http.Error(w, "unsupported resource", http.StatusNotImplemented)
		return
	}

	// Route to event handlers
	if len(parts) == 2 {
		// /calendars/{calendarId}/events
		switch r.Method {
		case http.MethodGet:
			s.listEvents(w, r, calendarID)
		case http.MethodPost:
			s.insertEvent(w, r, calendarID)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	} else if len(parts) == 3 {
		// /calendars/{calendarId}/events/{eventId}
		eventID := parts[2]
		switch r.Method {
		case http.MethodGet:
			s.getEvent(w, r, calendarID, eventID)
		case http.MethodPut, http.MethodPatch:
			s.updateEvent(w, r, calendarID, eventID)
		case http.MethodDelete:
			s.deleteEvent(w, r, calendarID, eventID)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	} else {
		http.Error(w, "invalid path", http.StatusBadRequest)
	}
}

// insertEvent handles POST /calendars/{calendarId}/events
func (s *Server) insertEvent(w http.ResponseWriter, r *http.Request, calendarID string) {
	var event calendar.Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate event ID
	event.Id = fmt.Sprintf("event%d", s.nextID)
	s.nextID++

	// Set metadata
	event.Status = "confirmed"
	event.Created = time.Now().Format(time.RFC3339)
	event.Updated = event.Created
	event.HtmlLink = fmt.Sprintf("https://calendar.google.com/event?eid=%s", event.Id)

	// Store event
	if s.events[calendarID] == nil {
		s.events[calendarID] = make(map[string]*calendar.Event)
	}
	s.events[calendarID][event.Id] = &event

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

// listEvents handles GET /calendars/{calendarId}/events
func (s *Server) listEvents(w http.ResponseWriter, r *http.Request, calendarID string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := r.URL.Query()
	timeMin := query.Get("timeMin")
	timeMax := query.Get("timeMax")
	maxResults := query.Get("maxResults")
	pageToken := query.Get("pageToken")
	singleEvents := query.Get("singleEvents")
	orderBy := query.Get("orderBy")

	// Get all events for calendar
	calEvents := s.events[calendarID]
	if calEvents == nil {
		calEvents = make(map[string]*calendar.Event)
	}

	// Convert to slice for filtering/sorting
	var events []*calendar.Event
	for _, evt := range calEvents {
		// Apply time filters
		if timeMin != "" && evt.Start != nil && evt.Start.DateTime != "" {
			if evt.Start.DateTime < timeMin {
				continue
			}
		}
		if timeMax != "" && evt.Start != nil && evt.Start.DateTime != "" {
			if evt.Start.DateTime > timeMax {
				continue
			}
		}
		events = append(events, evt)
	}

	// Sort events
	if orderBy == "startTime" && singleEvents == "true" {
		sort.Slice(events, func(i, j int) bool {
			iTime := ""
			jTime := ""
			if events[i].Start != nil {
				iTime = events[i].Start.DateTime
				if iTime == "" {
					iTime = events[i].Start.Date
				}
			}
			if events[j].Start != nil {
				jTime = events[j].Start.DateTime
				if jTime == "" {
					jTime = events[j].Start.Date
				}
			}
			return iTime < jTime
		})
	}

	// Handle pagination
	startIdx := 0
	if pageToken != "" {
		// Simple pagination: token is the start index
		fmt.Sscanf(pageToken, "%d", &startIdx)
	}

	maxRes := len(events)
	if maxResults != "" {
		fmt.Sscanf(maxResults, "%d", &maxRes)
	}

	endIdx := startIdx + maxRes
	if endIdx > len(events) {
		endIdx = len(events)
	}

	pagedEvents := events[startIdx:endIdx]

	// Build response
	resp := &calendar.Events{
		Kind:    "calendar#events",
		Summary: calendarID,
		Items:   pagedEvents,
	}

	// Add next page token if there are more results
	if endIdx < len(events) {
		resp.NextPageToken = fmt.Sprintf("%d", endIdx)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// getEvent handles GET /calendars/{calendarId}/events/{eventId}
func (s *Server) getEvent(w http.ResponseWriter, r *http.Request, calendarID, eventID string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	calEvents := s.events[calendarID]
	if calEvents == nil {
		http.Error(w, "calendar not found", http.StatusNotFound)
		return
	}

	event := calEvents[eventID]
	if event == nil {
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

// updateEvent handles PUT/PATCH /calendars/{calendarId}/events/{eventId}
func (s *Server) updateEvent(w http.ResponseWriter, r *http.Request, calendarID, eventID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	calEvents := s.events[calendarID]
	if calEvents == nil {
		http.Error(w, "calendar not found", http.StatusNotFound)
		return
	}

	existing := calEvents[eventID]
	if existing == nil {
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}

	var updates calendar.Event
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Preserve ID and metadata
	updates.Id = eventID
	updates.Created = existing.Created
	updates.Updated = time.Now().Format(time.RFC3339)
	updates.HtmlLink = existing.HtmlLink

	calEvents[eventID] = &updates

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updates)
}

// deleteEvent handles DELETE /calendars/{calendarId}/events/{eventId}
func (s *Server) deleteEvent(w http.ResponseWriter, r *http.Request, calendarID, eventID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	calEvents := s.events[calendarID]
	if calEvents == nil {
		http.Error(w, "calendar not found", http.StatusNotFound)
		return
	}

	if calEvents[eventID] == nil {
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}

	delete(calEvents, eventID)
	w.WriteHeader(http.StatusNoContent)
}

// Reset clears all events from the server.
func (s *Server) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = make(map[string]map[string]*calendar.Event)
	s.nextID = 1
}

// GetEvents returns all events for a calendar (for test assertions).
func (s *Server) GetEvents(calendarID string) []*calendar.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	calEvents := s.events[calendarID]
	if calEvents == nil {
		return nil
	}

	var events []*calendar.Event
	for _, evt := range calEvents {
		events = append(events, evt)
	}
	return events
}

// AddEvent adds a pre-configured event to the server (for test setup).
func (s *Server) AddEvent(calendarID string, event *calendar.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if event.Id == "" {
		event.Id = fmt.Sprintf("event%d", s.nextID)
		s.nextID++
	}

	if s.events[calendarID] == nil {
		s.events[calendarID] = make(map[string]*calendar.Event)
	}
	s.events[calendarID][event.Id] = event
}
