# Go Core Backend Engine

Production-grade, high-performance Backend Microservice Core built with Go (Golang) following Clean Architecture principles. Features enterprise-grade Zero-Trust Security, Granular RBAC, TOTP Two-Factor Authentication (2FA), Refresh Token Rotation, Prometheus metrics telemetry, token-bucket rate limiting, and an automated end-to-end test suite for Postman & Apidog.

[![Go CI & API Testing](https://github.com/Tnembull/go-core-backend/actions/workflows/ci.yml/badge.svg)](https://github.com/Tnembull/go-core-backend/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/Tnembull/go-core-backend)](https://goreportcard.com/report/github.com/Tnembull/go-core-backend)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## 🏛️ System Architecture

Built on Clean Architecture separation of concerns:

```
                      ┌─────────────────────────────────────────┐
                      │    HTTP Clients / Microservices / CDN   │
                      └────────────────────┬────────────────────┘
                                           │
                                           ▼
┌────────────────────────────────────────────────────────────────────────────────────────┐
│ Middlewares: Request ID Trace ➔ Slog Logger ➔ Prometheus Metrics ➔ IP Rate Limiter     │
└──────────────────────────────────────────┬─────────────────────────────────────────────┘
                                           │
                    ┌──────────────────────┴──────────────────────┐
                    ▼                                             ▼
       ┌────────────────────────┐                    ┌────────────────────────┐
       │ Public Handlers        │                    │ Protected Handlers     │
       │  - /healthz, /readyz   │                    │  - JWT Bearer (Access) │
       │  - /metrics            │                    │  - Granular RBAC       │
       │  - /auth/login         │                    │  - Machine-to-Machine  │
       │  - /auth/register      │                    │    (X-API-Key)         │
       └────────────┬───────────┘                    └────────────┬───────────┘
                    │                                             │
                    └──────────────────────┬──────────────────────┘
                                           ▼
                     ┌──────────────────────────────────────────┐
                     │ Service Layer (Core Business Logic)      │
                     │  - AuthService (JWT, TOTP 2FA, Rotation) │
                     │  - HealthService (Runtime Telemetry)     │
                     └─────────────────────┬────────────────────┘
                                           ▼
                     ┌──────────────────────────────────────────┐
                     │ Repository Layer (Thread-Safe Persistence│
                     │ & Revocation Token Blacklist)            │
                     └──────────────────────────────────────────┘
```

---

## 🛡️ Enterprise Security Features

### 0. Environment Profiles (Development vs Production)
The engine automatically shifts behavior based on `APP_ENV` (`development` vs `production`):

| Feature | Development Mode (`development`) | Production Mode (`production`) |
| :--- | :--- | :--- |
| **Configuration Loading** | Automatic `.env.local`, `.env.development`, `.env` | Environment Variables / Kubernetes Secrets |
| **Structured Logger** | Human-readable colored text, `DEBUG` level, `file:line` source | High-performance JSON, `INFO` level, RFC3339 timestamp |
| **CORS Policy** | Permissive wildcard `*` for local dev tools & vite | Strict origin domain whitelist (`CORS_ALLOWED_ORIGINS`) |
| **Runtime Profiler** | Active `net/http/pprof` mounted at `/debug/pprof` | Profiler disabled / quarantined behind auth |
| **Rate Limiter** | Lenient 200 RPS / 500 burst for rapid test iteration | Strict 50 RPS / 100 burst token bucket |
| **Telemetry Tagging** | Telemetry response reports `"environment": "development"` | Telemetry reports `"environment": "production"` |

### 1. Granular RBAC (Role-Based Access Control)
Supports hierarchical roles with fine-grained permission scopes:

| Role | Permissions | Accessible Scopes |
| :--- | :--- | :--- |
| `superadmin` | `*` (Wildcard) | Full system configuration & management |
| `admin` | `users:read`, `users:write`, `audit:read` | User administration & audit logs |
| `editor` | `users:read`, `users:write` | Content & user mutation |
| `viewer` | `users:read` | Read-only operations |

* Middleware filters:
  * `middleware.RequireRole("admin", "superadmin")`
  * `middleware.RequirePermission("users:write")`

### 2. Two-Factor Authentication (TOTP 2FA)
* Built on RFC 6238 TOTP (compatible with Google Authenticator, 1Password, Authy).
* **Setup flow**: `POST /api/v1/auth/2fa/setup` generates base32 secret, `otpauth://` QR URI, and 4 single-use recovery codes.
* **Activation**: `POST /api/v1/auth/2fa/enable` requires valid 6-digit TOTP code before activating 2FA.
* **Login flow with 2FA**:
  1. `POST /api/v1/auth/login` detects `two_factor_enabled: true` and issues a short-lived `2fa_preauth` temporary token (`mfa_required: true`).
  2. `POST /api/v1/auth/2fa/verify` verifies code + `temp_token` and grants full access session.

### 3. Token Rotation & Revocation
* Short-lived Access Token (HS256) paired with Long-lived Refresh Token (7 days).
* **Refresh**: `POST /api/v1/auth/refresh` rotates the token pair and blacklists the previous refresh token JTI.
* **Logout**: `POST /api/v1/auth/logout` invalidates the active refresh token.

### 4. Machine-to-Machine (M2M) Security
* Secure inter-service communication via `X-API-Key` authentication header (`GET /api/v1/internal/ping`).

---

## 🧪 Postman & Apidog Test Suite

The repository includes a 100% compliant **Postman Collection v2.1.0** test suite that runs seamlessly in both **Postman** and **Apidog**, as well as headlessly via **Newman CLI**.

* Collection file: `tests/postman/go-core-backend.postman_collection.json`
* Environment file: `tests/postman/go-core-backend.postman_environment.json`

### Executed Test Scenarios (20 Requests, 35 Assertions):
1. **Telemetry & Observability**:
   - `GET /healthz` (Liveness) & `GET /readyz` (Readiness)
   - `GET /metrics` (Prometheus telemetry validation)
   - `GET /api/v1/system/info` (Uptime, Memory, Goroutine inspect)
2. **Authentication & Token Lifecycle**:
   - Register Admin & Viewer users without password leakage
   - Login & automatic token propagation to Postman environment
   - Refresh Token exchange and rotation verification
3. **Granular RBAC Enforcement**:
   - Admin access granted to `/admin/dashboard` (200 OK)
   - Viewer denied access to `/admin/dashboard` (403 Forbidden)
   - Admin mutates data with `users:write` permission (200 OK)
   - Viewer denied mutation due to missing permission (403 Forbidden)
   - Admin denied system settings (requires superadmin `settings:manage`)
4. **TOTP 2FA Verification**:
   - Initiate 2FA setup and receive `otpauth://` URI
   - Rejection of invalid TOTP codes
5. **M2M Security & Revocation**:
   - Handshake with `X-API-Key`
   - Rejection of unauthorized internal requests (401)
   - Logout and verification that revoked refresh tokens cannot be reused

### Running Newman CLI Locally:
```bash
./tests/run_e2e_newman.sh
```

### Importing into Apidog / Postman:
1. Open **Apidog** or **Postman**.
2. Click **Import** -> Select `tests/postman/go-core-backend.postman_collection.json`.
3. Select Environment -> Import `tests/postman/go-core-backend.postman_environment.json`.
4. Run Collection -> All 35 assertions run and pass automatically.

---

## 🚀 Quickstart & Docker

### 1. Local Run
```bash
# Clone
git clone https://github.com/Tnembull/go-core-backend.git
cd go-core-backend

# Run Unit Tests
go test -v ./...

# Start Server
go run cmd/server/main.go
```

### 2. Multi-Stage Docker Container
```bash
# Build hardened non-root container
docker build -t go-core-backend:latest .

# Run container
docker run -d -p 8080:8080 --name core-api go-core-backend:latest
```

---

## 📄 License
MIT License. Authored by [Muhammad Nur Ashiddiqi (Tnembull)](https://github.com/Tnembull).
