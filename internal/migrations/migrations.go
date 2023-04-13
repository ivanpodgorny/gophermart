package migrations

import (
	"database/sql"
	"github.com/lopezator/migrator"
)

func Up(db *sql.DB) error {
	m, err := migrator.New(
		migrator.Migrations(
			&migrator.MigrationNoTx{
				Name: "Create users table",
				Func: createUsersTable,
			},
			&migrator.MigrationNoTx{
				Name: "Create tokens table",
				Func: createTokensTable,
			},
			&migrator.MigrationNoTx{
				Name: "Create orders table",
				Func: createOrdersTable,
			},
			&migrator.MigrationNoTx{
				Name: "Create transactions table",
				Func: createTransactionsTable,
			},
		),
	)
	if err != nil {
		return err
	}

	return m.Migrate(db)
}

func createUsersTable(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE users
(
    id            integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    login         varchar(20)  NOT NULL UNIQUE,
    password_hash varchar(100) NOT NULL
)
	`)

	return err
}

func createTokensTable(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE tokens
(
    id         integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    integer     NOT NULL REFERENCES users (id),
    token      varchar(32) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
)
	`)

	return err
}

func createOrdersTable(db *sql.DB) error {
	if _, err := db.Exec("CREATE TYPE order_status AS ENUM ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED')"); err != nil {
		return err
	}

	_, err := db.Exec(`
CREATE TABLE orders
(
    id          integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     integer     NOT NULL REFERENCES users (id),
    num         varchar(20) NOT NULL UNIQUE,
    status      order_status,
    accrual     real        NOT NULL DEFAULT 0,
    CHECK (accrual >= 0),
    uploaded_at timestamptz NOT NULL DEFAULT now()
)
	`)

	return err
}

func createTransactionsTable(db *sql.DB) error {
	if _, err := db.Exec("CREATE TYPE tx_type AS ENUM ('IN', 'OUT')"); err != nil {
		return err
	}

	if _, err := db.Exec(`
CREATE TABLE transactions
(
    id           integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id      integer     NOT NULL REFERENCES users (id),
    order_num    varchar(20) NOT NULL REFERENCES orders (num),
    type         tx_type     NOT NULL,
    amount       real        NOT NULL,
    CHECK (amount > 0),
    processed_at timestamptz NOT NULL DEFAULT now()
)
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
CREATE FUNCTION check_balance() RETURNS trigger AS
$$
DECLARE
    current_balance real;
BEGIN
    IF NEW.type = 'OUT' THEN
        current_balance := (SELECT coalesce(sum(amount), 0) FROM transactions WHERE user_id = NEW.user_id AND type = 'IN') -
                           (SELECT coalesce(sum(amount), 0) FROM transactions WHERE user_id = NEW.user_id AND type = 'OUT');
        IF NEW.amount > current_balance THEN
            RAISE 'Insufficient funds' USING ERRCODE = '23514';
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql
	`); err != nil {
		return err
	}

	_, err := db.Exec(`
CREATE TRIGGER check_balance
    BEFORE INSERT
    ON transactions
    FOR EACH ROW
EXECUTE FUNCTION check_balance()
	`)

	return err
}
