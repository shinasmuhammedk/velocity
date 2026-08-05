CREATE TABLE position_history (
    id UUID PRIMARY KEY,

    user_id BIGINT NOT NULL,

    symbol TEXT NOT NULL,

    quantity BIGINT NOT NULL,

    action VARCHAR(20) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_position_history_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_position_history_user
ON position_history(user_id);

CREATE INDEX idx_position_history_symbol
ON position_history(symbol);