# Architecture & Engineering Standards

## Overview
This repository enforces Clean Architecture principles for high-throughput Go microservices:
- **cmd/server**: Application entry point, dependency injection, and signal trapping.
- **internal/delivery**: HTTP handlers, routing with Chi router, and request/response serialization.
- **internal/service**: Domain business logic, token issuance, and validation rules.
- **internal/repository**: In-memory database persistence with mutex thread-safety.
- **pkg/response**: Standardized JSON envelopes and RFC 7807 error structures.

## Engineering Standards
All microservice components are verified through automated unit tests and end-to-end Newman Postman test suites.
