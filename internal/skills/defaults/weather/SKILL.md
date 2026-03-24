---
name: weather
description: "Retrieves current weather conditions, temperature, humidity, wind speed, and multi-day forecasts for any location worldwide using free public APIs (no API key required). Supports hourly and daily forecasts, rain and precipitation data, moon phases, and metric or imperial units. Use when the user asks about weather, temperature, forecast, rain, humidity, wind, tomorrow's weather, hourly conditions, or climate for a city or location."
version: 1.0.0
metadata:
  homepage: "https://wttr.in/:help"
---

# Weather

## Environment Variables

No environment variables or API keys required. This skill uses free public APIs:
- wttr.in (primary)
- Open-Meteo (fallback)

## Prerequisites

- `curl` command must be available in the system

Two free services, no API keys needed.

## wttr.in (primary)

Quick one-liner:
```bash
curl -s "wttr.in/London?format=3"
# Output: London: ⛅️ +8°C
```

Compact format:
```bash
curl -s "wttr.in/London?format=%l:+%c+%t+%h+%w"
# Output: London: ⛅️ +8°C 71% ↙5km/h
```

Full forecast:
```bash
curl -s "wttr.in/London?T"
```

Format codes: `%c` condition · `%t` temp · `%h` humidity · `%w` wind · `%l` location · `%m` moon

Tips:
- URL-encode spaces: `wttr.in/New+York`
- Airport codes: `wttr.in/JFK`
- Units: `?m` (metric) `?u` (USCS)
- Today only: `?1` · Current only: `?0`
- PNG: `curl -s "wttr.in/Berlin.png" -o /tmp/weather.png`

## Open-Meteo (fallback, JSON)

Free, no key, good for programmatic use:
```bash
curl -s "https://api.open-meteo.com/v1/forecast?latitude=51.5&longitude=-0.12&current_weather=true"
```

Find coordinates for a city, then query. Returns JSON with temp, windspeed, weathercode.

Docs: https://open-meteo.com/en/docs
