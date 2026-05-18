# afc-tools

Small Go CLI tools for Arsenal FC data.

Current command:

- `afc`

Current features:

- Fetch Arsenal results and fixtures from `arsenal.com`
- Fetch Arsenal fixtures and results from `football-data.org`
- Show detailed football-data.org match information by match id
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

Use `football-data.org` as the source:

```sh
AFC_FDAPI_TOKEN=your-token go run ./cmd/afc upcoming --source football-data
AFC_FDAPI_TOKEN=your-token go run ./cmd/afc past --source football-data
```

When using `football-data.org`, `upcoming` and `past` include match ids so they can
be used with `afc match <id>`.

Or pass the token as a flag:

```sh
go run ./cmd/afc upcoming --source football-data --football-data-token your-token
```

Match details by football-data.org match id:

```sh
AFC_FDAPI_TOKEN=your-token go run ./cmd/afc match 600001
```

Custom cache directory:

```sh
go run ./cmd/afc upcoming --cache-dir /tmp/afc-cache
```

## Cache

- Default cache directory: `~/.afctoolcache`
- Cache key for results/fixtures: `results-and-fixtures`
- Default cache freshness: 2 hours
- `football-data.org` responses use source-specific cache keys and shorter TTLs for live-ish data

## Notes

- Kickoff times are normalized from the visible Arsenal page kickoff text using
  the `Europe/London` timezone.
- This is necessary because the page's monthly table sometimes publishes
  incorrect UTC `datetime` values around DST boundaries.
