BEGIN;

CREATE SCHEMA IF NOT EXISTS gophermart;

CREATE TABLE IF NOT EXISTS gophermart.users (
    login varchar(32) NOT NULL,
    hash varchar(128) NOT NULL,
    salt varchar(16) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

ALTER TABLE gophermart.users ADD PRIMARY KEY(login);

CREATE INDEX IF NOT EXISTS login_idx ON gophermart.users USING hash(login);

COMMENT ON TABLE gophermart.users IS 'Таблица пользователей сервиса';

COMMENT ON COLUMN gophermart.users.login IS 'Логин пользователя';
COMMENT ON COLUMN gophermart.users.hash IS 'Зашифрованный пароль пользователя';
COMMENT ON COLUMN gophermart.users.salt IS 'Соль для хеширования пароля)';
COMMENT ON COLUMN gophermart.users.created_at IS 'Время создание пользователя';

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'order_status' AND typnamespace = (SELECT oid FROM pg_namespace WHERE nspname = 'gophermart')) THEN
        CREATE TYPE gophermart.order_status AS ENUM ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED');
    END IF;
END
$$;

CREATE TABLE IF NOT EXISTS gophermart.orders (
    id varchar(32) NOT NULL,
    status gophermart.order_status NOT NULL,
    money bigint DEFAULT 0,
    reward double precision DEFAULT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

ALTER TABLE gophermart.orders ADD PRIMARY KEY(id);

COMMENT ON TABLE gophermart.orders IS 'Таблица зазказов';

COMMENT ON COLUMN gophermart.orders.id IS 'Уникальный идентификатор заказов';
COMMENT ON COLUMN gophermart.orders.status IS 'Статус заказа';
COMMENT ON COLUMN gophermart.orders.money IS 'Сумма заказа в копейках';
COMMENT ON COLUMN gophermart.orders.reward IS 'Бонусное вознаграждение в копейках';
COMMENT ON COLUMN gophermart.orders.created_at IS 'Дата создания заказа';
COMMENT ON COLUMN gophermart.orders.updated_at IS 'Дата обновления заказа';

CREATE INDEX IF NOT EXISTS status_idx ON gophermart.orders USING btree(status);

COMMIT;
