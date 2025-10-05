# StravaCal

Syncs Strava club events to Google Calendar and generates a web schedule with embedded calendar. Auto-deploys to GitHub Pages every 15 minutes.

**Live Schedule**: [https://bkach.github.io/StravaCal/](https://bkach.github.io/StravaCal/)

**Add to Calendar**:
- [Subscribe in Google Calendar](https://calendar.google.com/calendar/u/0?cid=YjQ2YWVmMjA2OTQ1Njk0NDNhY2ZlZjUxZDllMTllNDEzYjE3YWRkZWIwMTkwZjljZWZkOGRhZDYzZWMzMGU3N0Bncm91cC5jYWxlbmRhci5nb29nbGUuY29t)
- [Download ICS File](https://bkach.github.io/StravaCal/calendar.ics)

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
go run .       # Full sync: Fetch from Strava → Google Calendar → HTML/ICS
go run . html  # Generate HTML only (fast, for testing web page)
go run . ics   # Generate ICS file only from cached events
go run . gcal  # Sync to Google Calendar only from cached events
go run . test  # Test with sample data from output/validation/events_raw.json
```

## GitHub Actions

Runs every 15 minutes. Set up:
1. Enable GitHub Pages (Settings → Pages → Source: GitHub Actions)
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
html.go     - HTML page generation with embedded Google Calendar
```

## Output

- `output/events/events.json` - Event data cache (all events from last 7 days)
- `output/schedules/index.html` - Web page with embedded Google Calendar
- `output/schedules/calendar.ics` - iCalendar file for calendar apps (next 60 days)

## Features

- **Google Calendar sync**: Automatically creates, updates, and deletes events in Google Calendar
- **Embedded calendar view**: Interactive Google Calendar with agenda view on web page
- **Automatic deployment**: GitHub Actions updates calendar every 15 minutes
- **ICS file generation**: Download calendar file for any calendar app
- **Phone number redaction**: Automatically removes phone numbers from event descriptions
- **Timezone handling**: All times properly converted to Europe/London (BST/GMT)
- **Event filtering**: Syncs next 60 days, caches last 7 days of events

## API

Uses `GET /api/v3/clubs/{id}/group_events?upcoming=true` (undocumented Strava endpoint)

**Rate limits**: 100 req/15min, 1000 req/day
