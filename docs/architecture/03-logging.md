# Logging Architecture

## Purpose

Logging provides visibility into the behavior of Velocity during development, testing, and production.

Logs help engineers understand request flow, detect failures, diagnose issues, and monitor system health.

The logging system must be lightweight, structured, and reusable.

---

# Goals

* Structured logging
* Consistent log format
* Low performance overhead
* Reusable across all modules
* Easy integration with monitoring tools

---

# Log Levels

Velocity uses the following log levels:

* Debug
* Info
* Warn
* Error
* Fatal

Each level should be used consistently across the application.

---

# Logging Principles

* Never use `fmt.Println()` for application logging.
* Every log entry should include contextual information.
* Errors should contain enough information for debugging.
* Sensitive information must never be logged.
* Logging should not affect request latency.

---

# Logging Flow

```text
Application Event
        │
        ▼
Logger
        │
        ▼
Structured Log Entry
        │
        ▼
Console / File
        │
        ▼
Monitoring System (Future)
```

---

# Logging Scope

The logger will be used by:

* Bootstrap
* HTTP Server
* Middleware
* Services
* Matching Engine
* Worker Pool
* Persistence Layer
* WebSocket Hub
* Recovery
* Monitoring

---

# Folder Ownership

```text
pkg/logger/
```

---

# Future Expansion

The logging system may later support:

* JSON logs
* Log rotation
* Correlation IDs
* Request IDs
* Trace IDs
* OpenTelemetry integration
* Centralized log aggregation
