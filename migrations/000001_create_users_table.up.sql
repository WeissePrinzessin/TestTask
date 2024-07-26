CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    passport_number VARCHAR(20) NOT NULL,
    surname VARCHAR(50) NOT NULL,
    name VARCHAR(50) NOT NULL,
    patronymic VARCHAR(50),
    address TEXT NOT NULL
);