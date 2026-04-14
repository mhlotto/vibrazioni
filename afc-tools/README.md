# afc-tools

Small Go CLI tools for Arsenal FC data.

Current command:

- `afc`

Current features:

- Fetch Arsenal results and fixtures from `arsenal.com`
- Parse the monthly results/fixtures list into structured match data
- Cache parsed data in `~/.afctoolcache`
- Show upcoming fixtures
- Show recent past results

## Build

```sh
go build ./cmd/afc
```

## Run

Upcoming fixtures for the next 7 days:

```sh
go run ./cmd/afc upcoming
```

Recent results from the last 7 days:

```sh
go run ./cmd/afc past
```

Custom day window:

```sh
go run ./cmd/afc upcoming --days 14
go run ./cmd/afc past --days 14
```

Custom cache directory:

```sh
go run ./cmd/afc upcoming --cache-dir /tmp/afc-cache
```

## Cache

- Default cache directory: `~/.afctoolcache`
- Cache key for results/fixtures: `results-and-fixtures`
- Default cache freshness: 2 hours

## Notes

- Kickoff times are normalized from the visible Arsenal page kickoff text using
  the `Europe/London` timezone.
- This is necessary because the page's monthly table sometimes publishes
  incorrect UTC `datetime` values around DST boundaries.
