package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	"example.com/devops-lab6/pharmacy/medicine"
	_ "github.com/lib/pq"
)

type postgres struct{ db *sql.DB }

func OpenPostgres(ctx context.Context) (*sql.DB, error) {
	host, port := os.Getenv("DB_HOST"), os.Getenv("DB_PORT")
	user, password, name := os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME")
	if host == "" || port == "" || user == "" || password == "" || name == "" {
		return nil, errors.New("DB_HOST, DB_PORT, DB_USER, DB_PASSWORD and DB_NAME are required")
	}

	address := url.URL{
		Scheme: "postgres",
		User: url.UserPassword(user, password),
		Host: net.JoinHostPort(host, port),
		Path: "/" + name,
		RawPath: "/" + url.PathEscape(name),
		RawQuery: "sslmode=disable&connect_timeout=3",
	}
	db, err := sql.Open("postgres", address.String())
	if err != nil {
		return nil, err
	}
	startup, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	if err := db.PingContext(startup); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("cannot connect to db: %w", err)
	}
	var migrated bool
	if err := db.QueryRowContext(startup, "SELECT to_regclass('public.medicines') IS NOT NULL").Scan(&migrated); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("check migrations: %w", err)
	}
	if !migrated {
		_ = db.Close()
		return nil, errors.New("medicines table is missing; run migrations")
	}
	return db, nil
}

func NewPostgresRepository(db *sql.DB) medicine.Repository {
	return &postgres{db: db}
}

func (r *postgres) Create(ctx context.Context, m medicine.Medicine) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO medicines (id, name, manufacturer, price, prescription, quantity)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		m.ID, m.Name, m.Manufacturer, m.Price, m.Prescription, m.Quantity)
	return err
}

func (r *postgres) GetByID(ctx context.Context, id string) (medicine.Medicine, error) {
	var m medicine.Medicine
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, manufacturer, price, prescription, quantity
		FROM medicines WHERE id = $1`, id).
		Scan(&m.ID, &m.Name, &m.Manufacturer, &m.Price, &m.Prescription, &m.Quantity)
	if errors.Is(err, sql.ErrNoRows) {
		return medicine.Medicine{}, medicine.ErrNotFound
	}
	return m, err
}

func (r *postgres) Sell(ctx context.Context, id string, count int) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE medicines SET quantity = quantity - $2
		WHERE id = $1 AND quantity >= $2`, id, count)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected > 0 {
		return err
	}

	var exists bool
	if err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM medicines WHERE id = $1)`, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return medicine.ErrNotFound
	}
	return medicine.ErrInsufficientStock
}
