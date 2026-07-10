# Database Architecture

## Purpose

PostgreSQL acts as the primary source of truth for all persistent data in Velocity.

## Technology

- PostgreSQL
- pgx/v5
- pgxpool
- SQLC
- golang-migrate

## Principles

- Database access occurs only through repositories.
- Business logic must never execute SQL directly.
- SQLC generates type-safe query code.
- Schema changes are managed through migrations.

## Flow

Handler
    ↓
Service
    ↓
Repository
    ↓
SQLC Generated Queries
    ↓
PostgreSQL

## Ownership

internal/persistence/postgres/
migrations/
sqlc.yaml