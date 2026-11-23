-- +goose Up
CREATE TABLE budgets (
    user_id  BIGINT NOT NULL REFERENCES users (id),
    category TEXT NOT NULL,
    monthly  BIGINT NOT NULL CHECK (monthly > 0),
    PRIMARY KEY (user_id, category)
);

CREATE TABLE budget_alerts (
    user_id   BIGINT NOT NULL REFERENCES users (id),
    category  TEXT NOT NULL,
    month     DATE NOT NULL,
    threshold SMALLINT NOT NULL CHECK (threshold IN (80, 100)),
    PRIMARY KEY (user_id, category, month, threshold)
);

-- +goose Down
DROP TABLE budget_alerts;
DROP TABLE budgets;
