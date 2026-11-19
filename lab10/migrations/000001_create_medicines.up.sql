CREATE TABLE medicines (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL CHECK (btrim(name) <> ''),
    manufacturer TEXT NOT NULL CHECK (btrim(manufacturer) <> ''),
    price DOUBLE PRECISION NOT NULL CHECK (price >= 0 AND price < 'Infinity'::double precision),
    prescription BOOLEAN NOT NULL,
    quantity BIGINT NOT NULL CHECK (quantity >= 0)
);
