CREATE TABLE wallet_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id BIGINT NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    asset TEXT NOT NULL,

    amount BIGINT NOT NULL,

    type VARCHAR(20) NOT NULL,

    trade_id BIGINT
        REFERENCES trades(id)
        ON DELETE SET NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT wallet_transactions_amount_positive
        CHECK (amount > 0),

    CONSTRAINT wallet_transactions_type_valid
        CHECK (
            type IN (
                'DEPOSIT',
                'WITHDRAWAL',
                'TRADE_CREDIT',
                'TRADE_DEBIT'
            )
        )
);

CREATE INDEX idx_wallet_transactions_user
ON wallet_transactions(user_id);

CREATE INDEX idx_wallet_transactions_user_asset
ON wallet_transactions(user_id, asset);

CREATE INDEX idx_wallet_transactions_created_at
ON wallet_transactions(created_at);

CREATE INDEX idx_wallet_transactions_trade
ON wallet_transactions(trade_id);

CREATE UNIQUE INDEX idx_wallet_transactions_trade_entry
ON wallet_transactions(
    trade_id,
    user_id,
    asset,
    type
)
WHERE trade_id IS NOT NULL;