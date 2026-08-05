
CREATE TABLE users (
    id BIGINT PRIMARY KEY ,

    email TEXT NOT NULL UNIQUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT users_email_lowercase
        CHECK (email = lower(email))
);