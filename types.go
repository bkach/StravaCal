package main

import "time"

// TokenStore holds the OAuth credentials for Strava API authentication
type TokenStore struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Event represents a standardized club event with all necessary information
// This is the main data structure used throughout the application
type Event struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Location    string    `json:"location"`
	Organizer   string    `json:"organizer"`
	SkillLevels *int      `json:"skill_levels,omitempty"` // 1=Beginner, 2=Intermediate, 4=Advanced
	Terrain     *int      `json:"terrain,omitempty"`      // 0=Road, 1=Trail, 2=Mixed
}

// StravaEvent represents the actual structure returned by the Strava API
// This was reverse-engineered from real API responses in October 2025
// Key differences from expected:
// - No start_date_local, uses upcoming_occurrences array instead
// - No end_date_local, must be calculated
// - Rich nested objects for organizer, route, etc.
type StravaEvent struct {
	ID                int64  `json:"id"`
	Title             string `json:"title"`
	Description       string `json:"description"`
	ClubID            int64  `json:"club_id"`
	OrganizingAthlete struct {
		ID        int64  `json:"id"`
		FirstName string `json:"firstname"`
		LastName  string `json:"lastname"`
	} `json:"organizing_athlete"`
	ActivityType        string    `json:"activity_type"` // e.g., "Run"
	RouteID             *int64    `json:"route_id"`      // May be null
	WomenOnly           bool      `json:"women_only"`
	Private             bool      `json:"private"`              // Always true for club events
	SkillLevels         *int      `json:"skill_levels"`         // 1=Beginner, 2=Intermediate, 4=Advanced
	Terrain             *int      `json:"terrain"`              // 0=Road, 1=Trail, 2=Mixed
	UpcomingOccurrences []string  `json:"upcoming_occurrences"` // ISO8601 timestamps
	Zone                string    `json:"zone"`                 // e.g., "Europe/London"
	Address             string    `json:"address"`              // Location description or coordinates
	Joined              bool      `json:"joined"`               // If current user joined
	StartLatLng         []float64 `json:"start_latlng"`         // [lat, lng] coordinates
}

// TokenResponse represents the response from Strava OAuth token endpoint
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}
