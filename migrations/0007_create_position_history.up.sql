CREATE TABLE position_history (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    symbol TEXT NOT NULL,
    quantity BIGINT NOT NULL,
    action VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_position_history_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);