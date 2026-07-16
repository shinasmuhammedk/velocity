# Configuration Management

## Purpose

The configuration system centralizes all application settings and removes hard-coded values from the codebase.

It allows Velocity to run in different environments without code changes.

---

# Goals

* Environment-independent configuration
* Validation during startup
* Centralized configuration access
* Type-safe configuration
* Easy deployment

---

# Configuration Sources

Velocity loads configuration from:

1. YAML configuration files
2. Environment variables
3. Default values (where appropriate)

Environment variables always override file values.

---

# Configuration Flow

```text
Application Starts
        │
        ▼
Load YAML
        │
        ▼
Load Environment Variables
        │
        ▼
Merge Configuration
        │
        ▼
Validate Configuration
        │
        ▼
Fail Fast On Errors
        │
        ▼
Expose Immutable Configuration
        │
        ▼
Expose Configuration
```

---

# Configuration Categories

Examples include:

* Application
* HTTP Server
* PostgreSQL
* JWT
* Logging
* Metrics
* Redis (future)
* Kafka/NATS (future)

---

# Design Principles

* No hard-coded configuration
* Immutable after startup
* Loaded once
* Shared across the application
* Fail Fast if invalid
* Strong startup validation

---

# Folder Ownership

```text
configs/

internal/config/
```

---

# Future Expansion

Future configuration may include:

* Distributed deployment
* Kubernetes secrets
* Cloud storage
* External service configuration
* Feature flags
* Snapshot configuration
* Recovery configuration
* WebSocket configuration
* Risk engine configuration
* Persistence worker configuration
* Event store configuration
