# StravaCal

Automatically syncs Strava club events to Google Calendar. Runs every 15 minutes via GitHub Actions.

## 📅 View the Calendar

**Google Calendar (Recommended)**: [View Calendar](https://calendar.google.com/calendar/u/0/embed?src=b46aef20694569443acfef51d9e19e413b17addeb0190f9cefd8dad63ec30e77@group.calendar.google.com&ctz=Europe/London)
- Bookmark this link for easy access
- No subscription needed, just view events directly

**Subscribe to Calendar**:
- Google Calendar: [Add to your Google Calendar](https://calendar.google.com/calendar/u/0?cid=YjQ2YWVmMjA2OTQ1Njk0NDNhY2ZlZjUxZDllMTllNDEzYjE3YWRkZWIwMTkwZjljZWZkOGRhZDYzZWMzMGU3N0Bncm91cC5jYWxlbmRhci5nb29nbGUuY29t)
- Other calendar apps: [Download ICS file](https://bkach.github.io/StravaCal/calendar.ics) and import into your calendar app

## Setup

Set environment variables:
```bash
export STRAVA_CLIENT_ID="your_client_id"
export STRAVA_CLUB_ID="your_club_id"
export CLIENT_SECRET="your_client_secret"
export REFRESH_TOKEN="your_refresh_token"
export GOOGLE_CALENDAR_ID="your_calendar_id"
```

For Google Calendar sync, also provide:
```bash
export GOOGLE_SERVICE_ACCOUNT="your_service_account_json"
# OR place service-account.json in the project root
```

## Commands

```bash
go run .       # Full sync: Fetch from Strava → Google Calendar → ICS
go run . ics   # Generate ICS file only from cached events
go run . gcal  # Sync to Google Calendar only from cached events
go run . test  # Test with sample data from output/validation/events_raw.json
```

## GitHub Actions

Runs every 15 minutes to sync events. Set up:
1. Enable GitHub Pages (Settings → Pages → Source: GitHub Actions) - Only needed for ICS file hosting
2. Add secrets:
   - `STRAVA_CLIENT_ID`
   - `STRAVA_CLUB_ID`
   - `CLIENT_SECRET`
   - `REFRESH_TOKEN`
   - `GOOGLE_CALENDAR_ID`
   - `SERVICE_ACCOUNT_B64` (base64 encoded service account JSON)

## Project Structure

```
main.go     - Entry point and command handling
types.go    - Shared data structures
strava.go   - Strava API integration (OAuth, event fetching, phone number redaction)
gcal.go     - Google Calendar sync (create, update, delete events)
ics.go      - ICS calendar file generation (RFC 5545 format)
```

## Output

- `output/events/events.json` - Event data cache (all events from last 7 days)
- `output/calendar.ics` - iCalendar file for calendar apps (next 60 days)

## Features

- **Google Calendar sync**: Automatically creates, updates, and deletes events in Google Calendar
- **Automatic updates**: GitHub Actions syncs calendar every 15 minutes
- **ICS file generation**: Downloadable calendar file for any calendar app
- **Phone number redaction**: Automatically removes phone numbers from event descriptions
- **Timezone handling**: All times properly converted to Europe/London (BST/GMT)
- **Event filtering**: Syncs next 60 days, caches last 7 days of events
- **Smart sync**: Only updates changed events, removes deleted ones

## API

Uses `GET /api/v3/clubs/{id}/group_events?upcoming=true` (undocumented Strava endpoint)

**Rate limits**: 100 req/15min, 1000 req/day
