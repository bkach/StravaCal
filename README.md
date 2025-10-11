# StravaCal

Automatically syncs Strava club events to Google Calendar. Runs every 15 minutes via GitHub Actions.

## 🏃 Buzzard Running Club Calendar

Stay up to date with all upcoming Malvern Buzzard runs.

### 📅 View the Calendar (Recommended)

**Google Calendar Embed:**  
👉 [Open Buzzard Run Schedule](https://calendar.google.com/calendar/u/0/embed?src=b46aef20694569443acfef51d9e19e413b17addeb0190f9cefd8dad63ec30e77@group.calendar.google.com&ctz=Europe/London)

- Bookmark this link for quick access  
- No subscription or login required — view events directly in your browser  

---

### 🔔 Subscribe to the Calendar

**Add to Your Google Calendar:**
[+ Add Buzzard Runs to Google Calendar](https://calendar.google.com/calendar/u/0?cid=YjQ2YWVmMjA2OTQ1Njk0NDNhY2ZlZjUxZDllMTllNDEzYjE3YWRkZWIwMTkwZjljZWZkOGRhZDYzZWMzMGU3N0Bncm91cC5jYWxlbmRhci5nb29nbGUuY29t)

### 🗓️ Use Another Calendar App

If you're **not using Google Calendar**, you can still stay up to date by subscribing to the live `.ics` feed.

#### 🍎 Apple Calendar (Mac / iPhone / iPad)
1. Copy this link:
   ```
   https://bkach.github.io/StravaCal/calendar.ics
   ```

2. Open the **Calendar app**
3. In the menu bar, go to **File → New Calendar Subscription...**
4. Paste the link above and click **Subscribe**
5. (Optional) Change the name, color, and refresh frequency (e.g., *every hour*)

This will keep your Buzzard Runs calendar automatically updated.

#### 💼 Outlook (Desktop or Web)
- Go to your Calendar view
- Select **Add Calendar → Subscribe from Web**
- Paste the same URL:
  ```
  https://bkach.github.io/StravaCal/calendar.ics
  ```
- Confirm to add and sync automatically

#### 🖥️ Other Calendar Apps
Most calendar apps (like Thunderbird, Zoho, or Fastmail) have an option to **"Subscribe to calendar by URL."**
Use the same `.ics` link:
```
https://bkach.github.io/StravaCal/calendar.ics
```

---

🪄 **Tip:** If you *import* the file manually (instead of subscribing), it will **not update automatically** when new runs are added. Always use the subscription method if possible.

---

## ⚙️ Setup – Use This for Your Own Club

Want to generate a live calendar like this for your own Strava club? Do so by cloning this repo and following these (admittedly very sparse) instructions.

### Required Environment Variables

These are always required:
```bash
export STRAVA_CLIENT_ID="your_strava_client_id"
export STRAVA_CLUB_ID="your_strava_club_id"
export CLIENT_SECRET="your_strava_client_secret"
export REFRESH_TOKEN="your_strava_refresh_token"
```

### Optional: Google Calendar Sync

To sync to Google Calendar (optional for ICS-only generation), add:
```bash
export GOOGLE_CALENDAR_ID="your_google_calendar_id"
```

And provide service account credentials (choose one method):
```bash
# Method 1: Environment variable (raw JSON string)
export GOOGLE_SERVICE_ACCOUNT='{"type":"service_account","project_id":"...",...}'

# Method 2: File (recommended for local development)
# Place service-account.json in the project root
```

## Commands

```bash
go run .       # Full sync: Fetch from Strava → Google Calendar → ICS
go run . ics   # Generate ICS file only from cached events
go run . gcal  # Sync to Google Calendar only from cached events
go run . test  # Test with sample data from output/validation/events_raw.json
```

## GitHub Actions

Runs every 15 minutes to sync events to Google Calendar and generate ICS file. See [`.github/workflows/update-calendar.yml`](.github/workflows/update-calendar.yml) for the workflow configuration.

Set up:
1. Enable GitHub Pages (Settings → Pages → Source: GitHub Actions) - Only needed if you want to host the ICS file publicly
2. Add repository secrets (Settings → Secrets and variables → Actions):
   - `STRAVA_CLIENT_ID` - Your Strava OAuth client ID
   - `STRAVA_CLUB_ID` - Your Strava club ID
   - `CLIENT_SECRET` - Your Strava OAuth client secret
   - `REFRESH_TOKEN` - Your Strava OAuth refresh token
   - `GOOGLE_CALENDAR_ID` - Target Google Calendar ID
   - `SERVICE_ACCOUNT_B64` - Base64 encoded Google service account JSON (see workflow for how it's decoded)

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
