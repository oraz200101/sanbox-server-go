CREATE TABLE charm
(
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT UNIQUE,
    description TEXT
);

CREATE TABLE customers
(
    id         TEXT PRIMARY KEY,
    name       TEXT,
    surname    TEXT,
    patronymic TEXT,
    gender     TEXT,
    birth_date DATE,
    charm_id   TEXT,
    cia_id     TEXT
);

CREATE INDEX idx_customers_charm_id ON customers (charm_id);

CREATE TABLE customer_address
(
    customer_id TEXT NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    type        TEXT,
    street      TEXT,
    house       TEXT,
    apartment   TEXT,
    PRIMARY KEY (customer_id, type)
);

CREATE INDEX idx_customer_address_customer_id ON customer_address (customer_id);

CREATE TABLE customer_phone
(
    number      TEXT,
    customer_id TEXT NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    type        TEXT,
    PRIMARY KEY (number, customer_id)
);

CREATE INDEX idx_customer_phone_customer_id ON customer_phone (customer_id);