# wwlp-tools

`wwlp-tools` is a small Go command line tool for reading WWLP site data and
formatting it for terminal use.

Current commands:

- `headlines`
- `headline-lists`
- `weather`
- `alerts`

The `weather` command also supports a `clothes` mode that sends a weather
summary to a llama.cpp server through its OpenAI-compatible API and asks for a
what-to-wear recommendation.

## Requirements

- Go 1.21 or newer
- `make`

## Build

Default build:

```sh
make build
```

This creates:

```sh
bin/wwlp
```

The default llama.cpp host is injected at build time. The source code does not
hardcode a private network address.

Build with a custom default llama.cpp host:

```sh
make build DEFAULT_LLAMA_HOST=192.168.16.19
```

Other useful build variables:

- `BINARY` default: `wwlp`
- `BIN_DIR` default: `bin`
- `CGO_ENABLED` default: `0`
- `GOFLAGS` default: empty

## Test

Run package tests:

```sh
GOCACHE=/tmp/wwlp-go-build go test ./pkg/wwlp
```

Run all tests:

```sh
GOCACHE=/tmp/wwlp-go-build go test ./...
```

Note: some older tests in this repo expect fixture files that are not currently
present in the checkout:

- `natlang-forecast.out`
- `natlang-forecast-desiredblob.json`
- `weather-alerts-response.json`

Because of that, `go test ./...` may fail even when the clothing-mode code is
working correctly.

## Basic Usage

Show command help:

```sh
bin/wwlp
```

List top headlines:

```sh
bin/wwlp headlines
```

Show weather forecast discussion:

```sh
bin/wwlp weather
```

Show current conditions:

```sh
bin/wwlp weather current
```

Show seven-day forecast:

```sh
bin/wwlp weather seven-day
```

Show weather alerts from the weather service:

```sh
bin/wwlp weather alerts
```

## Clothes Mode

Default clothes recommendation:

```sh
bin/wwlp weather clothes
```

This mode:

1. Loads WWLP weather data.
2. Builds a short weather summary.
3. Sends that summary to llama.cpp using `/v1/chat/completions`.
4. Prints the clothing recommendation.

Supported built-in profiles:

- `mid40-male`
- `mid40-female`
- `teen`
- `young-adult-male`
- `young-adult-female`
- `senior`

Example with profile and custom notes:

```sh
bin/wwlp weather clothes \
  --profile senior \
  --activity "walking to the bus stop" \
  --profile-notes "gets cold easily"
```

Ask for a short poem after the clothing advice:

```sh
bin/wwlp weather clothes --poem
```

Ask for playful, catty delivery:

```sh
bin/wwlp weather clothes --sassy
bin/wwlp weather clothes --sassy --poem
```

Print the exact prompts sent to llama.cpp:

```sh
bin/wwlp weather clothes --debug-prompt
```

## llama.cpp Settings

Defaults used by clothes mode:

- Host: build-time value from `DEFAULT_LLAMA_HOST`, fallback `127.0.0.1`
- Port: `8080`
- Model: `Qwen/Qwen3-4B-GGUF`

You can override them at runtime:

```sh
bin/wwlp weather clothes \
  --llama-host 192.168.18.13 \
  --llama-port 8080 \
  --model Qwen/Qwen3-4B-GGUF
```

The llama.cpp server must expose an OpenAI-compatible chat completions API at:

```text
http://HOST:PORT/v1/chat/completions
```

## Input Options

Most commands can read from:

- the live WWLP endpoint by default
- a local file with `--file PATH`
- standard input with `--file -`

Examples:

```sh
bin/wwlp weather --file saved-template-vars.json --mode current
bin/wwlp headlines --file -
```

## Clean

Remove build output:

```sh
make clean
```
