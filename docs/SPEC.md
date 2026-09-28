# Project Specification

Brazilian Capitals Weather API — functional and technical specification.

| | |
| --- | --- |
| **Status** | Implemented, verified |
| **Version** | 1.0.0 |
| **Machine-readable contract** | [`api/openapi.yaml`](../api/openapi.yaml) |
| **Provenance** | Written 100% by Qwen3.8-Max (Qwen Code CLI) |

---

## 1. Overview

A read-only HTTP API that reports the current air temperature — plus feels-like
temperature, humidity, wind speed and sky condition — for the 27 Brazilian state
capitals. Data is fetched live from Open-Meteo on every request.

The project's second purpose is architectural: it is a reference implementation
of hexagonal architecture with strict SOLID adherence in Go, small enough to
read end to end.

## 2. Goals

| # | Goal |
| --- | --- |
| G1 | Serve live temperature for all 27 capitals through a single business endpoint. |
| G2 | Allow clients to narrow the response to a subset of capitals. |
| G3 | Keep the domain free of infrastructure concerns, so the upstream provider is replaceable. |
| G4 | Run with no secrets, no database and no runtime dependencies. |
| G5 | Fail loudly and honestly: never render a wrong number as if it were right. |

## 3. Non-goals

| # | Explicitly out of scope |
| --- | --- |
| N1 | Forecasts, historical series, hourly or daily aggregation. |
| N2 | Caching, retries, circuit breaking. |
| N3 | Authentication, authorisation, rate limiting. |
| N4 | Municipalities that are not state capitals. |
| N5 | Persistence. The capital dataset is embedded in the binary. |
| N6 | Alternative units (Fahrenheit), i18n of the response, content negotiation. |
| N7 | Write operations of any kind. The API is read-only. |

## 4. Functional requirements

| ID | Requirement |
| --- | --- |
| FR-01 | `GET /v1/capitals/weather` returns the current weather for all 27 capitals when no filter is supplied. |
| FR-02 | The optional `capital` query parameter restricts the response to the named capitals. |
| FR-03 | `capital` accepts a comma-separated list in one parameter and the same slug repeated across multiple parameters; both may be mixed. |
| FR-04 | Slug matching is case-insensitive and treats spaces and underscores as hyphens. |
| FR-05 | Duplicate slugs within one request collapse to a single reading. |
| FR-06 | Blank slug entries (empty or whitespace-only) are ignored rather than rejected. |
| FR-07 | Unfiltered results are ordered alphabetically by capital name. Filtered results preserve the order in which the slugs were requested. |
| FR-08 | Each reading carries identity (`slug`, `name`, `state_code`, coordinates), temperature, feels-like temperature, humidity, wind speed, WMO condition code and description, and the observation instant. |
| FR-09 | `observed_at` is rendered with the observing capital's own UTC offset, not a single national offset. |
| FR-10 | Every response names its data source (`source.name`, `source.url`) to satisfy the upstream attribution requirement. |
| FR-11 | A request naming an unknown slug fails with 400 and reports both the invalid slugs and the full list of valid ones. |
| FR-12 | A request naming more than 50 slugs fails with 400 before any upstream call. |
| FR-13 | `GET /healthz` reports process liveness without touching any dependency. |
| FR-14 | Unknown paths fail with a JSON 404; known paths with a wrong method fail with a JSON 405 carrying an `Allow` header. |
| FR-15 | Every response body, success or failure, is JSON with `Content-Type: application/json; charset=utf-8`. |
| FR-16 | The `capitals` array serialises as `[]`, never `null`, when there are no readings. |
| FR-17 | Upstream failures map to 502; upstream timeouts to 504. Neither leaks the upstream's internal message to the client. |
| FR-18 | A panic inside a handler is recovered, logged with a stack trace, and answered as a JSON 500. The process stays up. |
| FR-19 | `SIGINT` and `SIGTERM` trigger a graceful shutdown: stop accepting, drain in-flight requests up to `SHUTDOWN_TIMEOUT`, exit 0. |
| FR-20 | Internal error causes are logged server-side and replaced with a generic client-facing message. |

## 5. Non-functional requirements

| ID | Requirement | Target | Verified |
| --- | --- | --- | --- |
| NFR-01 | Latency, unfiltered (27 capitals) | < 3 s p50 | 0.91 s |
| NFR-02 | Latency, filtered (2 capitals) | < 1 s p50 | 0.36 s |
| NFR-03 | Upstream calls per request | exactly 1 | 1 (batched) |
| NFR-04 | Statement coverage over `./internal/...` | ≥ 90 % | 98.8 % |
| NFR-05 | Data race freedom | clean under `-race` | clean |
| NFR-06 | Static analysis | `go vet`, `gofmt` clean | clean |
| NFR-07 | Runtime third-party dependencies | 0 | 0 (`go.uber.org/mock` is test-only) |
| NFR-08 | Startup failure on invalid configuration | fail fast, non-zero exit | verified in `config` tests |
| NFR-09 | Bounded resource use | response body read capped | 1 MiB upstream read cap, 50-slug request cap |
| NFR-10 | Portability | Go 1.24+, no cgo, no tzdata required | offsets come from the upstream payload |
| NFR-11 | Observability | one structured log line per request | method, path, query, status, bytes, duration, remote address |

## 6. Domain model

```
Capital (entity, immutable)
  Slug, Name, StateCode, Latitude, Longitude

CurrentWeather (value object)
  Capital, TemperatureCelsius, FeelsLikeCelsius, HumidityPercent,
  WindSpeedKmh, ConditionCode, ObservedAt
  Condition() -> string          derived from ConditionCode, never stored

WeatherCode (value type)
  WMO 4677 code table
  Description() -> string        falls back to "Uncatalogued condition"
  Known()       -> bool
```

Invariant: `CurrentWeather.Condition()` is a pure function of `ConditionCode`.
The description is never stored, so the two cannot disagree.

Invariant: an unknown WMO code degrades to a generic description instead of
failing, so a future code published by the WMO cannot break the API.

### Domain errors

| Error | Meaning | Maps to |
| --- | --- | --- |
| `ErrCapitalNotFound` (sentinel) | A slug matched no capital. | 400 |
| `*UnknownCapitalsError` | Carries the invalid slugs and all valid ones; unwraps to `ErrCapitalNotFound`. | 400 `unknown_capital` |
| `*InvalidRequestError` | An input rule was broken (payload too large). | 400 `invalid_request` |
| `ErrWeatherUnavailable` (sentinel) | The upstream failed. Wraps the original cause. | 502 `upstream_failure` |

## 7. Port contracts

The core declares these; adapters implement them.

### `CapitalRepository`

```go
All() []Capital                        // alphabetical; returns a copy
BySlug(slug string) (Capital, bool)    // slug must already be normalised
```

### `WeatherProvider`

```go
CurrentWeather(ctx context.Context, capitals []Capital) ([]CurrentWeather, error)
```

Any implementation must, to remain substitutable:

1. return exactly one `CurrentWeather` per capital received, **in the same order**;
2. never return an error for valid capitals — network, contract and upstream
   failures are errors;
3. honour context cancellation and deadlines;
4. tolerate an empty input without contacting the upstream.

Batching belongs to the contract because it expresses a domain capability
("fetch the weather for these capitals"), not a provider optimisation. An
implementation without batch support may fan out internally.

### `Clock`

```go
Now() time.Time
```

Exists solely so use cases can be tested against fixed instants.

### `WeatherQuery` (declared by the driving adapter)

```go
Execute(ctx context.Context, slugs []string) (*application.Result, error)
```

Declared by `rest`, the consumer, rather than by `application` — the idiomatic
Go direction for interfaces, and what makes the handler testable with a double.

## 8. Use case behaviour

`GetCapitalsWeather.Execute(ctx, slugs)`:

1. `slugs` empty → resolve to every capital.
2. `len(slugs) > 50` → `*InvalidRequestError`, no upstream call.
3. Normalise each slug; drop blanks; drop repeats (first occurrence wins).
4. Resolve each against the repository; collect misses.
5. Any miss → `*UnknownCapitalsError` with the full valid-slug list, no
   upstream call.
6. Call the provider once with the resolved capitals.
7. Provider error → wrap with `%w` around `ErrWeatherUnavailable`, keeping the
   original cause reachable via `errors.Is`.
8. Stamp `RetrievedAt` from the `Clock`, converted to UTC.

The caller's `context` is passed through untouched, so cancellation and request
scoping reach the upstream socket.

## 9. Upstream integration contract

Provider: Open-Meteo `GET /v1/forecast`. No API key.

Request parameters: `latitude` and `longitude` as comma-separated lists,
`current=temperature_2m,apparent_temperature,relative_humidity_2m,wind_speed_10m,weather_code`,
`timezone=auto`.

### UP-01 — two payload shapes

Open-Meteo returns a bare JSON **object** for one coordinate and an **array**
for several. The adapter decodes through `json.RawMessage`, trying the array
first and falling back to the object, then asserts the entry count equals the
number of capitals requested.

### UP-02 — batch integrity

The upstream documents order preservation, but the adapter does not rely on it.
Each entry's coordinates are compared against the capital it is paired with,
within **0.5°**. Rationale: São Paulo and Rio de Janeiro are ~0.65° apart, so a
swapped pair is detected, while grid snapping (typically < 0.2°) is tolerated.
A violation is a hard error, because the alternative is silently attributing one
city's temperature to another.

### UP-03 — required versus optional fields

`temperature_2m` is required; its absence fails the request rather than
reporting `0`. `apparent_temperature`, `relative_humidity_2m`, `wind_speed_10m`
and `weather_code` are optional and degrade to zero, with an absent
`weather_code` becoming `-1` (uncatalogued).

### UP-04 — timezone strategy (measured)

Brazil spans four UTC offsets, so a single named zone renders wrong local times
for Rio Branco (`-05:00`) and for Manaus, Boa Vista, Porto Velho, Campo Grande
and Cuiabá (`-04:00`). `timezone=auto` resolves each location's own zone — and
is also dramatically faster. Measured over three runs of the 27-coordinate batch:

| `timezone` | Run 1 | Run 2 | Run 3 |
| --- | --- | --- | --- |
| `America/Sao_Paulo` | timeout at 30 s | timeout at 30 s | 22.5 s |
| `auto` | 1.43 s | 0.72 s | 0.72 s |
| *(omitted — UTC)* | 0.73 s | 0.74 s | 0.72 s |
| parallel fan-out, 27 requests | 0.69 s wall | 0.63 s wall | — |

Consequences adopted:

- `OPEN_METEO_TIMEZONE` defaults to `auto`. Setting it to a named IANA zone
  re-introduces both the wrong-offset bug and the 20–30 s latency.
- The batch is kept over the fan-out: comparable latency, but 1 upstream call
  instead of 27, which matters against a free-tier rate limit of roughly
  600 calls/min.
- Timestamps are parsed with `time.FixedZone` built from the payload's own
  `utc_offset_seconds`, so no tzdata database is needed in the container.

### UP-05 — failure classification

| Upstream condition | Adapter error |
| --- | --- |
| Non-200 status | `ErrUnexpectedStatus`, message carries the provider's `reason` or the raw body |
| `{"error": true, "reason": …}` even with status 200 | `ErrMalformedPayload` |
| Unparsable or empty body | `ErrMalformedPayload` |
| Missing `current` block or missing temperature | `ErrMalformedPayload` |
| Unparsable `current.time` | `ErrMalformedPayload` |
| Entry count ≠ capitals requested | `ErrMismatchedBatch` |
| Coordinates outside the 0.5° tolerance | `ErrMismatchedBatch` |
| Transport failure (dial, DNS, TLS, timeout, body read) | wrapped cause, `context.DeadlineExceeded` preserved so the transport layer can answer 504 |

## 10. Architecture constraints

| ID | Constraint |
| --- | --- |
| AC-01 | `internal/domain` imports only the standard library. |
| AC-02 | `internal/application` imports `internal/domain` and nothing else in this module. |
| AC-03 | No package under `internal` imports an adapter package, except adapters themselves. |
| AC-04 | A driving adapter must not import a driven adapter. Cross-adapter data (such as source attribution) is carried by the composition root. |
| AC-05 | `cmd/api/main.go` is the only file that names concrete adapter types. |
| AC-06 | `internal/config` is the only package that reads the environment. |
| AC-07 | `internal/mocks` is generated. It is never edited by hand and is regenerated with `make mocks`. |
| AC-08 | Every response leaves through the same JSON envelope writer. |

## 11. Acceptance criteria

| ID | Criterion | Covered by |
| --- | --- | --- |
| AC-T01 | All 27 capitals resolve, alphabetically, with unique canonical slugs, unique state codes and coordinates inside Brazil. | `memory` dataset tests |
| AC-T02 | `All()` returns a copy; callers cannot corrupt the dataset. | `memory` tests |
| AC-T03 | Every declared WMO constant has a description. | `domain` invariant test |
| AC-T04 | Slug normalisation is idempotent and handles case, spaces, underscores. | `domain` tests |
| AC-T05 | Filtering, ordering, de-duplication and blank handling behave per FR-03…FR-07. | `application` tests (gomock) |
| AC-T06 | Unknown slugs and oversized requests fail before any upstream call. | `application` tests — strict doubles fail on an unexpected provider call |
| AC-T07 | Provider failures wrap into `ErrWeatherUnavailable` with the cause intact. | `application` tests |
| AC-T08 | The caller's context reaches the provider. | `application` tests |
| AC-T09 | Both upstream payload shapes decode correctly. | `openmeteo` tests |
| AC-T10 | Every UP-05 failure classification is exercised. | `openmeteo` tests |
| AC-T11 | A reordered batch is rejected rather than silently misattributed. | `openmeteo` tests |
| AC-T12 | `context.DeadlineExceeded` survives transport wrapping. | `openmeteo` tests |
| AC-T13 | The full response envelope matches the OpenAPI schema, including `[]` for empty results. | `rest` tests |
| AC-T14 | The whole error matrix (400/404/405/500/502/503/504) returns the right status, code and headers. | `rest` tests |
| AC-T15 | A panicking handler yields a JSON 500 without leaking the panic value. | `rest` tests |
| AC-T16 | The configured request timeout becomes a context deadline on the use case call. | `rest` tests |
| AC-T17 | Configuration defaults, overrides and fail-fast validation. | `config` tests |
| AC-T18 | `WriteTimeout` always exceeds `RequestTimeout`. | `config` tests |
| AC-T19 | Graceful shutdown drains in-flight requests and exits 0. | manual run: `SIGTERM` → exit 0, in-flight request completed |
| AC-T20 | Live end-to-end against the real provider. | manual run: 27 capitals in 0.91 s, three distinct UTC offsets present |

## 12. Verified behaviour

Recorded from a live run against `api.open-meteo.com` on 2026-09-27:

```
GET /v1/capitals/weather                        -> 200, count 27, 0.91 s
GET /v1/capitals/weather?capital=sao-paulo,curitiba -> 200, count 2, 0.36 s
GET /v1/capitals/weather?capital=campinas       -> 400 unknown_capital
POST /v1/capitals/weather                       -> 405 method_not_allowed, Allow: GET, HEAD
GET /nope                                       -> 404 not_found
GET /healthz                                    -> 200 {"status":"ok"}
SIGTERM                                         -> exit 0, in-flight request drained
```

Distinct UTC offsets observed in one response: `-03:00`, `-04:00`, `-05:00`,
confirming FR-09.

## 13. Future work

Ordered by value if the API is ever productionised:

1. **Decorate `WeatherProvider` with a short-TTL cache.** Open-Meteo refreshes
   `current` every 15 minutes, so a 60 s TTL would cut upstream load with no loss
   of fidelity. The port is already the right seam; no other layer changes.
2. **Retry with jitter on 5xx and timeouts**, plus a circuit breaker, to absorb
   the upstream latency variance documented in UP-04.
3. **A readiness probe** distinct from `/healthz`, so orchestration can tell
   "process alive" from "can serve traffic".
4. **Move the capital dataset behind a real repository** if it ever needs to
   change without a redeploy.
5. **Contract tests against the live provider** in CI, scheduled rather than
   per-commit, to catch upstream schema drift.
6. **Request IDs** propagated from a header into logs and the upstream call.
