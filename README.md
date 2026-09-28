# Brazilian Capitals Weather API

A single-endpoint Go HTTP API that returns the **live** temperature and weather
condition for all 27 Brazilian state capitals (26 states plus the Federal
District), built on a hexagonal architecture with strict SOLID adherence.

> **Provenance** — this project was written **100% by [Qwen3.8-Max](https://qwen.ai)**
> through the Qwen Code CLI: architecture, source code, tests, OpenAPI
> specification, documentation and build tooling. No hand-written code.

---

## Highlights

- **One business endpoint**, no API key required, **zero runtime dependencies** —
  standard library only. [`go.uber.org/mock`](https://github.com/uber-go/mock)
  is the sole external module and it is test-only.
- **Hexagonal architecture** with a dependency rule enforced by package layout:
  the domain imports nothing, adapters import inward, and only `cmd/api` knows
  concrete types.
- **Live data, no cache.** Every request performs exactly one batched upstream
  call to [Open-Meteo](https://open-meteo.com/).
- **98.8% statement coverage** across `./internal/...`, all tests green under
  `-race`, `go vet` and `gofmt` clean.
- Structured JSON logging (`log/slog`), panic recovery, graceful shutdown on
  `SIGINT`/`SIGTERM`, and request-scoped timeouts that propagate all the way to
  the upstream socket.

---

## Quick start

Requires Go 1.24 or newer.

```bash
make run                      # listens on :8080
curl -s localhost:8080/v1/capitals/weather | jq
```

Or compile a binary. Build and coverage artifacts are written **outside** the
repository — to `$TMPDIR/sample-test-project/` — so no target ever adds files to
the working tree:

```bash
make build       # -> $TMPDIR/sample-test-project/api
make cover       # -> $TMPDIR/sample-test-project/coverage.out
make clean       # remove them
```

Filter to specific capitals:

```bash
curl -s 'localhost:8080/v1/capitals/weather?capital=sao-paulo,curitiba' | jq
```

All targets:

```bash
make help    # list targets
make check   # gofmt + go vet + go test -race
make cover   # cross-package coverage report
make mocks   # regenerate the gomock mocks
```

---

## API reference

The machine-readable contract lives in [`api/openapi.yaml`](api/openapi.yaml).

### `GET /v1/capitals/weather`

Returns the current weather for every capital, or for the ones requested.

| Query parameter | Required | Description |
| --- | --- | --- |
| `capital` | no | Comma-separated slugs (`?capital=sao-paulo,curitiba`) or repeated (`?capital=sao-paulo&capital=curitiba`). Omit for all 27. Max 50 entries. |

Slugs are case-insensitive; spaces and underscores are treated as hyphens, so
`Sao_Paulo`, `SAO PAULO` and `sao-paulo` all resolve. Duplicates collapse.

Without a filter, capitals come back alphabetically by name. With a filter, they
come back in the order you asked for.

**200 OK**

```json
{
  "count": 2,
  "retrieved_at": "2026-09-27T23:45:12.481Z",
  "source": { "name": "Open-Meteo", "url": "https://open-meteo.com/" },
  "capitals": [
    {
      "slug": "rio-branco",
      "name": "Rio Branco",
      "state_code": "AC",
      "latitude": -9.9754,
      "longitude": -67.8249,
      "temperature_c": 27.3,
      "feels_like_c": 30.1,
      "humidity_percent": 74,
      "wind_speed_kmh": 5.2,
      "condition": "Partly cloudy",
      "condition_code": 2,
      "observed_at": "2026-09-27T19:45:00-05:00"
    },
    {
      "slug": "sao-paulo",
      "name": "São Paulo",
      "state_code": "SP",
      "latitude": -23.5505,
      "longitude": -46.6333,
      "temperature_c": 18.9,
      "feels_like_c": 19.2,
      "humidity_percent": 86,
      "wind_speed_kmh": 9.7,
      "condition": "Overcast",
      "condition_code": 3,
      "observed_at": "2026-09-27T21:45:00-03:00"
    }
  ]
}
```

`observed_at` carries each capital's own UTC offset — note Rio Branco at `-05:00`
and São Paulo at `-03:00` describing the same instant. `capitals` is always an
array, never `null`. `condition_code` is `-1` if the upstream omitted it.

**Available slugs** (27)

```
aracaju        belem              belo-horizonte  boa-vista     brasilia
campo-grande   cuiaba             curitiba        florianopolis fortaleza
goiania        joao-pessoa        macapa          maceio        manaus
natal          palmas             porto-alegre    porto-velho   recife
rio-branco     rio-de-janeiro     salvador        sao-luis      sao-paulo
teresina       vitoria
```

### `GET /healthz`

Liveness probe. Touches no dependency, so `{"status":"ok"}` means only that the
process is up and accepting connections — it is deliberately **not** a readiness
check for the upstream.

### Errors

Every failure is JSON with a stable machine-readable `code`:

| Status | `code` | When |
| --- | --- | --- |
| 400 | `unknown_capital` | A slug matched no capital. `details` lists `invalid_slugs` and all `valid_slugs`. |
| 400 | `invalid_request` | More than 50 slugs in one request. |
| 404 | `not_found` | Unknown path. |
| 405 | `method_not_allowed` | Known path, wrong method. Includes an `Allow` header. |
| 502 | `upstream_failure` | Open-Meteo errored, or returned a payload that broke its contract. |
| 503 | `request_canceled` | The client went away mid-request. |
| 504 | `upstream_timeout` | The upstream did not answer within `REQUEST_TIMEOUT`. |
| 500 | `internal_error` | Unexpected failure or recovered panic. The cause is logged, never disclosed. |

```bash
curl -s 'localhost:8080/v1/capitals/weather?capital=campinas'
```

```json
{
  "error": {
    "code": "unknown_capital",
    "message": "capital not found: campinas",
    "details": {
      "invalid_slugs": ["campinas"],
      "valid_slugs": ["aracaju", "belo-horizonte", "belem", "..."]
    }
  }
}
```

Internal error details never reach the client — a panic value or a wrapped
driver error is logged server-side and replaced with a generic message.

---

## Configuration

Everything is environment-driven; there are no config files and no flags.
`internal/config` is the only package that reads the environment.

| Variable | Default | Description |
| --- | --- | --- |
| `HTTP_ADDR` | `:8080` | Listen address. |
| `REQUEST_TIMEOUT` | `10s` | Per-request budget, applied as a `context` deadline that reaches the upstream socket. |
| `SHUTDOWN_TIMEOUT` | `10s` | Grace period for in-flight requests after `SIGTERM`. |
| `OPEN_METEO_BASE_URL` | `https://api.open-meteo.com` | Upstream host. Point it at a stub to run offline. |
| `OPEN_METEO_TIMEOUT` | `8s` | HTTP client timeout for the upstream call. |
| `OPEN_METEO_TIMEZONE` | `auto` | See [the timezone finding](#the-timezone-finding) — do not set this to a named IANA zone without reading it. |
| `SOURCE_NAME` | `Open-Meteo` | Attribution rendered in `source.name`. |
| `SOURCE_URL` | `https://open-meteo.com/` | Attribution rendered in `source.url`. |
| `LOG_LEVEL` | `info` | Any `slog` level: `debug`, `info`, `warn`, `error`. |
| `LOG_FORMAT` | `json` | `json` or `text`. |

Durations use Go syntax (`10s`, `1m30s`). Invalid or non-positive values fail
fast at startup rather than silently falling back to a default.

```bash
HTTP_ADDR=:9090 LOG_LEVEL=debug LOG_FORMAT=text make run
```

---

## Architecture

Hexagonal (ports and adapters). The domain is the core; everything else is a
plug.

```
                        ┌──────────────────────────────────────────┐
   HTTP request ──────► │  DRIVING ADAPTER                         │
                        │  internal/adapter/driving/rest           │
                        │  router · handler · DTO · error mapping  │
                        └───────────────────┬──────────────────────┘
                                            │ depends on
                        ┌───────────────────▼──────────────────────┐
                        │  APPLICATION                             │
                        │  internal/application                    │
                        │  GetCapitalsWeather (the only use case)  │
                        └───────────────────┬──────────────────────┘
                                            │ depends on
                  ┌─────────────────────────▼─────────────────────────┐
                  │  DOMAIN  ·  internal/domain                       │
                  │                                                   │
                  │  Capital · CurrentWeather · WeatherCode           │
                  │                                                   │
                  │  PORTS (interfaces the core owns):                │
                  │    CapitalRepository   Clock                      │
                  │    WeatherProvider                                │
                  │                                                   │
                  │  imports nothing but the standard library         │
                  └─────────────────────────▲─────────────────────────┘
                                            │ implemented by
                        ┌───────────────────┴──────────────────────┐
                        │  DRIVEN ADAPTERS                         │
                        │  internal/adapter/driven/memory          │
                        │  internal/adapter/driven/openmeteo       │
                        └───────────────────┬──────────────────────┘
                                            │
                                            ▼
                                    Open-Meteo HTTP API
```

**The dependency rule:** source code dependencies point inward only. `domain`
imports nothing. `application` imports `domain`. Adapters import `domain` (and
`application`, for driving adapters). Nothing imports an adapter.

**Dependency inversion in practice:** the ports are declared by the *consumer*.
`domain.WeatherProvider` is defined in the core and implemented by
`openmeteo.Client`; the core has no idea Open-Meteo exists. Likewise
`rest.WeatherQuery` is declared by the HTTP layer and satisfied by
`application.GetCapitalsWeather`.

`cmd/api/main.go` is the **composition root** — the only file allowed to name
concrete types. It wires the object graph and owns the server lifecycle.

### Project layout

```
cmd/api/main.go                          composition root, wiring, lifecycle
api/openapi.yaml                         OpenAPI 3.1 contract
docs/SPEC.md                             functional and technical specification
internal/
  domain/                                CORE — no imports outside stdlib
    capital.go                           Capital entity
    weather.go                           CurrentWeather value object
    weather_code.go                      WMO 4677 table and descriptions
    clock.go                             Clock port + SystemClock
    errors.go                            sentinel and structured errors
    slug.go                              NormalizeSlug
    ports.go                             CapitalRepository, WeatherProvider
  application/
    get_capitals_weather.go              the use case
  adapter/
    driving/rest/                        inbound HTTP
      router.go                          routing, 404/405 policy
      weather_handler.go                 handler, query parsing, error mapping
      dto.go                             response envelope
      middleware.go                      request log, panic recovery
    driven/openmeteo/                    outbound HTTP
      client.go                          provider client
      dto.go                             upstream payload shapes
    driven/memory/
      capital_repository.go              the 27-capital dataset
  config/config.go                       environment to typed settings
  logging/logging.go                     slog construction
  mocks/                                 generated by mockgen (do not edit)
```

---

## SOLID

| Principle | Where it shows up |
| --- | --- |
| **S**ingle responsibility | Each type has one reason to change. `WeatherHandler` translates HTTP↔DTO and nothing else; the use case orchestrates and never touches `net/http`; `openmeteo.Client` owns the provider protocol; `config` is the only reader of the environment. |
| **O**pen/closed | A new provider means implementing `domain.WeatherProvider` — the use case, handler and DTOs stay untouched. A new WMO code means adding one map entry; `Description()` falls back gracefully instead of failing, so an uncatalogued code cannot break a response. |
| **L**iskov substitutability | The `WeatherProvider` contract is documented on the interface (one reading per capital, same order, no error for valid input, honours cancellation). Every implementation — `openmeteo.Client` and the gomock double — is interchangeable without the use case noticing. |
| **I**nterface segregation | Ports are as small as possible: `WeatherProvider` and `Clock` have one method each, `CapitalRepository` has two, and no consumer depends on a method it does not call. `openmeteo.Doer` narrows `*http.Client` to the single method actually used. |
| **D**ependency inversion | `application` and `domain` depend only on abstractions they own. Concrete types are injected in `main.go`. `config.Load` even takes a `Lookup` function so it is testable without touching process environment. |

Two deliberate layering choices worth calling out:

- **Naming.** Domain types speak the business language and adapter types speak
  the technical one, so `domain.CurrentWeather` sits next to `rest.WeatherHandler`
  without either naming convention leaking into the other layer. Capital *names*
  stay in Portuguese (`São Paulo`, `Goiânia`) because they are data, not code.
- **Attribution is injected, not imported.** The `source` block names Open-Meteo,
  but `rest` receives it as a value from `main.go`. A driving adapter importing a
  driven adapter would collapse the hexagon, so the composition root carries it
  across instead.

---

## Testing

```bash
make test    # go test ./internal/... -race
make cover   # 98.8% of statements across ./internal/...
```

10 test files, 52 test functions, ~1630 lines of tests against ~1220 lines of
source.

| Layer | Technique |
| --- | --- |
| `domain` | Pure unit tests. Includes an invariant test that every declared WMO constant has a description, so a new code cannot be merged half-wired. |
| `application` | gomock doubles for `CapitalRepository`, `WeatherProvider` and `Clock`. Asserts resolution order, de-duplication after normalisation, blank-slug handling, the 50-slug ceiling, error wrapping, and that the `context` reaches the provider. Because the doubles are strict, an unexpected call — such as hitting the provider after slug resolution failed — fails the test on its own. |
| `adapter/driven/memory` | Dataset integrity: 27 entries, alphabetical, unique and canonical slugs, unique two-letter state codes, coordinates inside Brazil's bounding box, and `All()` returning a copy rather than internal state. |
| `adapter/driven/openmeteo` | `httptest.Server` for protocol behaviour (both payload shapes, malformed JSON, error envelopes, batch-size and coordinate mismatches, grid snapping) and a gomock `Doer` for transport failures that need no socket (dial refused, DNS, TLS, timeout, body read error). |
| `adapter/driving/rest` | `httptest` through the real router with a mocked `WeatherQuery`. Covers the full JSON envelope, every filter syntax, timeout propagation, the whole error matrix, panic recovery, and 404/405 routing. |
| `config`, `logging` | Table-driven, with an injectable `Lookup` and an injectable writer. |

### Mocks

Generated by [`mockgen`](https://github.com/uber-go/mock) into `internal/mocks`.
**Do not edit them.** `mockgen` is pinned as a `go.mod` tool dependency, so no
global install is needed:

```bash
make mocks          # regenerate
go tool mockgen …   # what make runs
```

Mocks live in one package rather than beside each interface so that the domain
tree stays free of test-tooling imports.

---

## Design notes

### The timezone finding

`OPEN_METEO_TIMEZONE` defaults to `auto`, and that is load-bearing on two counts.

Batching all 27 capitals against a **named** IANA zone (`America/Sao_Paulo`)
measured at **22–30 s** per call, frequently exceeding a 30 s curl timeout. The
same batch with `timezone=auto` measured at **0.7–1.4 s** — roughly 30× faster.
Measured over three runs each:

| `timezone` | Run 1 | Run 2 | Run 3 |
| --- | --- | --- | --- |
| `America/Sao_Paulo` | timeout (30 s) | timeout (30 s) | 22.5 s |
| `auto` | 1.43 s | 0.72 s | 0.72 s |
| *(omitted, UTC)* | 0.73 s | 0.74 s | 0.72 s |

It is also a correctness issue, not just a performance one. Brazil spans four
UTC offsets, so pinning every capital to São Paulo time renders wrong local
times for Rio Branco (`-05:00`) and for Manaus, Boa Vista, Porto Velho, Campo
Grande and Cuiabá (`-04:00`). With `auto`, each reading carries its own offset.

A parallel fan-out of 27 single-coordinate requests measured at 0.63–0.69 s wall
time — comparable to the batch, but it costs 27 upstream calls instead of 1 and
would exhaust Open-Meteo's free-tier rate limit after roughly 20 API requests
per minute. The batch was kept for that reason.

### Batch integrity check

Open-Meteo preserves request order, but the adapter does not take that on faith.
Each returned entry's coordinates are checked against the capital it is being
paired with, within a 0.5° tolerance — tight enough that swapping São Paulo and
Rio de Janeiro (~0.65° apart) is caught, loose enough for the upstream's grid
snapping (typically under 0.2°). Without it, a reordered batch would silently
attribute one city's temperature to another, with no error and no visible
symptom.

### Two payload shapes

Open-Meteo returns a bare JSON **object** for a single coordinate and an
**array** for several. Decoding goes through `json.RawMessage` and tries the
array first, falling back to the object, so a one-capital filter does not break.
This is the most common way integrations with this API fail, and it is covered by
a dedicated test.

### Other decisions

- **No cache, deliberately.** The requirement was live data. Open-Meteo refreshes
  its `current` block every 15 minutes, so a short TTL cache would cut upstream
  load without any loss of fidelity — it is the first thing to add if rate limits
  ever bite, and `WeatherProvider` is the seam to decorate.
- **Temperature is required; everything else is optional.** A missing
  `temperature_2m` fails the request rather than silently reporting `0`. Missing
  humidity, wind or weather code degrade to zero and `Uncatalogued condition`,
  because none of them is what the endpoint promises.
- **Error mapping is centralised.** The handler maps domain sentinels to statuses
  in one `switch`, and `context.DeadlineExceeded` is checked *before*
  `ErrWeatherUnavailable` so a timeout surfaces as 504 rather than a generic 502.
- **Routing without a catch-all.** Every response, including 404 and 405, uses
  the same JSON envelope. Registering a `/` pattern cannot achieve that: it
  matches wrong-method requests too and hides the 405, while the mux's built-in
  405 answers in plain text. Paths are matched against an explicit route table
  that also supplies the `Allow` header, so the mux never has to guess.
- **`WriteTimeout` is derived**, not fixed: `REQUEST_TIMEOUT + 5s`. A write
  deadline shorter than the request budget would let the server close the
  connection while the handler is still waiting on the upstream.

---

## Limitations

- **Upstream latency is variable.** Open-Meteo's free tier has no SLA. The 8 s
  client timeout and 10 s request budget absorb normal variance, but a slow
  upstream will surface as a 504. There is no retry and no cache to mask it.
- **No rate limiting or authentication.** Suitable for internal use; put a
  gateway in front before exposing it publicly.
- **The capital dataset is embedded.** Changing it means a redeploy. Swapping in
  a database requires only a new `domain.CapitalRepository` implementation.
- **`cmd/api/main.go` is not unit-tested.** It is pure wiring whose only logic is
  blocking on signals; it was instead verified by running the binary and
  observing a clean `SIGTERM` shutdown (exit code 0, in-flight request drained).
- **Two defensive branches are unreachable** — `json.Marshal` failures in the
  response writer. They are kept because silently dropping an encode error would
  be worse than an untested line.

---

## Attribution

Weather data by [Open-Meteo](https://open-meteo.com/), under
[CC BY 4.0](https://creativecommons.org/licenses/by/4.0/). The `source` block in
every response carries the attribution at runtime, as their terms require.

Coordinates are the municipal seats of each capital. WMO condition descriptions
follow the [WMO 4677](https://open-meteo.com/en/docs) code table.

## License

MIT.
