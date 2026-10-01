# Go Core Backend Engine

Production-grade, high-performance Backend Microservice Core built with Go (Golang) following Clean Architecture principles, featuring automated Prometheus metrics, token-bucket rate limiting, JWT & API Key authentication, structured logging, graceful shutdown, and a **complete Postman & Apidog automated test suite**.

---

## 🎯 Architecture Overview

```text
                               +----------------------------------+
                               |        HTTP Client Request       |
                               +-----------------+----------------+
                                                 |
                                                 v
                       +--------------------------------------------------+
                       |           Core Middleware Pipeline               |
                       |  - Request ID (UUID Trace Injection)             |
                       |  - Structured Logging (slog JSON)                |
                       |  - Prometheus Metrics (Counter & Histogram)      |
                       |  - Token Bucket IP Rate Limiting                 |
                       |  - CORS & Panic Recovery                         |
                       +-------------------------+------------------------+
                                                 |
                                                 v
                       +--------------------------------------------------+
                       |               Chi HTTP Router                    |
                       +---------+---------------+--------------+---------+
                                 |               |              |
                                 v               v              v
                       +-----------------+ +------------+ +---------------+
                       |  Health Handler | | Auth Layer | | Users / Admin |
                       +-----------------+ +------------+ +---------------+
                                 |               |              |
                                 v               v              v
                       +--------------------------------------------------+
                       |             Clean Service Domain                 |
                       |  - Password Hashing (Bcrypt)                     |
                       |  - JWT Claims Token Generation & Validation      |
                       |  - RBAC Scope Guard (admin vs user)              |
                       +-------------------------+------------------------+
                                                 |
                                                 v
                       +--------------------------------------------------+
                       |         Repository & Persistence Layer           |
                       +--------------------------------------------------+
```

---

## 🚀 Key Features

1. **Clean Layered Architecture**:
   - `internal/handler`: HTTP request parsing and response orchestration.
   - `internal/service`: Domain business rules and security policies.
   - `internal/repository`: Data access interfaces and thread-safe persistence.
   - `internal/middleware`: Modular request pipeline (Tracing, Metrics, Rate limiting, Auth).
   - `pkg/response`: Standardized JSON envelope format with `trace_id` and timestamps.

2. **Full Observability & Kubernetes Probes**:
   - `GET /healthz`: Liveness probe for Kubernetes pod lifecycle.
   - `GET /readyz`: Readiness probe ensuring dependencies are healthy before taking traffic.
   - `GET /metrics`: Standard Prometheus metrics scraping endpoint (`http_requests_total`, `http_request_duration_seconds`).
   - `GET /api/v1/system/info`: Live runtime memory diagnostics and server uptime.

3. **Multi-Vector Security & RBAC**:
   - **Bcrypt (Cost 10)** password hashing.
   - **JWT (HMAC-SHA256)** authentication with configurable expiration.
   - **Role-Based Access Control (RBAC)** guarding admin-only endpoints.
   - **Machine-to-Machine Authentication** via static `X-API-Key` headers.
   - **In-Memory Token Bucket Rate Limiter** to prevent DoS attacks.

4. **Reliability & Graceful Shutdown**:
   - Listens for `SIGINT` / `SIGTERM` signals.
   - Drains active connections with a 10-second graceful shutdown window.

---

## 🧪 Postman & Apidog Automated Testing Suite

This repository includes a production-ready API test suite compatible with **Postman**, **Apidog**, and the **Newman CLI**.

### 1. File Locations
- **Collection**: `tests/postman/go-core-backend.postman_collection.json`
- **Environment**: `tests/postman/go-core-backend.postman_environment.json`
- **Automated Runner**: `tests/run_e2e_newman.sh`

### 2. How to Import to Apidog / Postman GUI
1. Open **Postman** or **Apidog**.
2. Click **Import** -> Select `tests/postman/go-core-backend.postman_collection.json`.
3. Import `tests/postman/go-core-backend.postman_environment.json` into Environments.
4. Run the Collection runner: all 11 endpoints and 22 assertions will automatically validate:
   - Dynamic token extraction and propagation into protected endpoints (`{{jwt_token}}`).
   - Response status codes (200, 201, 401).
   - Trace ID header presence.
   - Metrics payload integrity.

### 3. Run Automated CLI Test (Newman)
```bash
./tests/run_e2e_newman.sh
```

---

## 📁 Repository Structure

```text
├── cmd/
│   └── server/
│       └── main.go                     # Application entrypoint & dependency injection
├── internal/
│   ├── config/                         # Environment configuration parser
│   ├── handler/                        # HTTP controllers (Health, Auth, User)
│   ├── middleware/                     # Trace ID, Logger, Metrics, Rate Limit, Auth
│   ├── model/                          # Domain models, DTOs, and JWT Claims
│   ├── repository/                     # Data access interface & thread-safe store
│   └── service/                        # Business logic & security hashing
├── pkg/
│   ├── logger/                         # Structured slog JSON logger
│   └── response/                       # Standardized JSON response envelope
├── tests/
│   ├── postman/                        # Postman & Apidog Collections and Environments
│   └── run_e2e_newman.sh               # Automated Newman CLI test executor
├── Dockerfile                          # Multi-stage hardened Alpine container
└── .github/workflows/
    └── ci.yml                          # Go tests + Newman Postman runner in CI
```

---

## 🛠️ Quickstart

### 1. Run Locally
```bash
# Clone the repository
git clone https://github.com/Tnembull/go-core-backend.git
cd go-core-backend

# Copy environment template
cp .env.example .env

# Run server
go run cmd/server/main.go
```

### 2. Run with Docker
```bash
docker build -t go-core-backend .
docker run -p 8080:8080 go-core-backend
```

### 3. Run Unit & Integration Tests
```bash
go test -v ./...
```

---

## 🛡️ Security & Zero Leak Posture
- All production secrets, database credentials, and JWT keys are isolated into `.env` (guarded by `.gitignore`).
- Docker container runs strictly as unprivileged user `appuser:10001` with minimal attack surface.

---
**Author**: Muhammad Nur Ashiddiqi  
**Portfolio**: [muhammadnurashiddiqi.my.id](https://muhammadnurashiddiqi.my.id)
