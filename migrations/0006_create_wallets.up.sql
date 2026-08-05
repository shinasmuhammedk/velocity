CREATE TABLE wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id BIGINT NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    asset TEXT NOT NULL,

    available BIGINT NOT NULL DEFAULT 0,

    locked BIGINT NOT NULL DEFAULT 0,

    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT wallets_available_non_negative
        CHECK (available >= 0),

    CONSTRAINT wallets_locked_non_negative
        CHECK (locked >= 0),

    UNIQUE(user_id, asset)
);

CREATE INDEX idx_wallets_user
ON wallets(user_id);

CREATE INDEX idx_wallets_asset
ON wallets(asset);