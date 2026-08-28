CREATE TABLE users(
    tg_bot_id Bigint PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    domain VARCHAR(255) NOT NULL,
    current_price DECIMAL(10, 2) NOT NULL DEFAULT 0,
    last_checked_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE subscriptions (
    user_id Bigint REFERENCES users(tg_bot_id) ON DELETE CASCADE,
    product_id INT REFERENCES products(id) ON DELETE CASCADE,
    target_price DECIMAL(10, 2) NOT NULL,
    PRIMARY KEY(user_id, product_id)
);
