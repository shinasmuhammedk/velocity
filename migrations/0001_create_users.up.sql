-- 0001_create_users.up.sql
-- Users: authentication identity. Referenced by orders, trades, positions.

CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()

CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT users_email_unique     UNIQUE (email),
    CONSTRAINT users_email_lowercase  CHECK (email = lower(email))
);

-- users_email_unique already provisions a btree index on email for login lookups;
-- no separate index needed.