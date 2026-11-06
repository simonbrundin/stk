// Package store är ett beständigt lager för kopplingen mellan det
// kommunala bokningssystemet och MATCHi.
//
// Databasen är en Neon-databas (PostgreSQL) som skapas automatiskt vid
// uppstart via EnsureSchema, så att API:t kan köras utan extern
// migrationsordning.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Store är en wrapper runt en PostgreSQL-anslutning.
type Store struct {
	db *sql.DB
}

// CourtMapping kopplar en resurs i det kommunala bokningssystemet till en
// bana eller sal i MATCHi.
type CourtMapping struct {
	// @Description	Resursens id i det kommunala bokningssystemet
	// @Example		628
	KommunResourceID int64 `json:"kommun_resource_id"`

	// @Description	Resursens namn i det kommunala bokningssystemet
	// @Example		Baldershallen 1
	KommunResourceName string `json:"kommun_resource_name"`

	// @Description	MATCHis id för banan eller salen
	// @Example		3622
	MatchiCourtID int64 `json:"matchi_court_id"`

	// @Description	MATCHis namn för banan eller salen
	// @Example		1. Tennis - Vasa Bryggeri
	MatchiCourtName string `json:"matchi_court_name"`
}

// Booking är en bokning som speglas mellan det kommunala bokningssystemet
// och MATCHi.
type Booking struct {
	// @Description	Id i det kommunala bokningssystemet
	// @Example		351930
	KommunAppointmentID int64 `json:"kommun_appointment_id"`

	// @Description	Resursens id i det kommunala bokningssystemet
	// @Example		631
	KommunResourceID int64 `json:"kommun_resource_id"`

	// @Description	MATCHis id för banan eller salen
	// @Example		3622
	MatchiCourtID int64 `json:"matchi_court_id"`

	// @Description	MATCHis numeriska boknings-id
	// @Example		112291396
	MatchiBookingID string `json:"matchi_booking_id,omitempty"`

	// @Description	MATCHis slot-id för bokningen
	// @Example		4a66b2f0-3177-4c9c-a7a5-3a54e44a9737
	MatchiSlotID string `json:"matchi_slot_id,omitempty"`

	// @Description	Starttid i lokal tid
	// @Example		2026-11-06T11:00:00+01:00
	StartTime time.Time `json:"start_time"`

	// @Description	Sluttid i lokal tid
	// @Example		2026-11-06T13:00:00+01:00
	EndTime time.Time `json:"end_time"`

	// @Description	Status för bokningen, t.ex. synced
	// @Example		synced
	Status string `json:"status"`

	// @Description	Tidpunkten när raden skapades
	// @Example		2026-11-06T10:00:00Z
	CreatedAt time.Time `json:"created_at"`

	// @Description	Tidpunkten när raden senast ändrades
	// @Example		2026-11-06T10:00:00Z
	UpdatedAt time.Time `json:"updated_at"`
}

// New skapar en store mot en PostgreSQL-databas.
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// Open öppnar en anslutning mot en PostgreSQL-databas.
func Open(dsn string) (*Store, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open database: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	return New(db), nil
}

// Close stänger anslutningen.
func (s *Store) Close() error {
	return s.db.Close()
}

// Ping kontrollerar att databasen svarar.
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// schemaSQL beskriver tabellerna som används.
const schemaSQL = `
CREATE TABLE IF NOT EXISTS court_mappings (
    id                   BIGSERIAL PRIMARY KEY,
    kommun_resource_id   BIGINT NOT NULL,
    kommun_resource_name TEXT NOT NULL DEFAULT '',
    matchi_court_id      BIGINT NOT NULL,
    matchi_court_name    TEXT NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (kommun_resource_id, matchi_court_id)
);

CREATE TABLE IF NOT EXISTS bookings (
    id                     BIGSERIAL PRIMARY KEY,
    kommun_appointment_id  BIGINT NOT NULL,
    kommun_resource_id     BIGINT NOT NULL,
    matchi_court_id        BIGINT NOT NULL,
    matchi_booking_id      TEXT NOT NULL DEFAULT '',
    matchi_slot_id         TEXT NOT NULL DEFAULT '',
    start_time             TIMESTAMPTZ NOT NULL,
    end_time               TIMESTAMPTZ NOT NULL,
    status                 TEXT NOT NULL DEFAULT 'synced',
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (kommun_appointment_id, matchi_court_id)
);

CREATE INDEX IF NOT EXISTS idx_bookings_start_time ON bookings (start_time);
CREATE INDEX IF NOT EXISTS idx_bookings_matchi_slot_id ON bookings (matchi_slot_id);
`

// EnsureSchema skapar tabellerna om de inte redan finns.
func (s *Store) EnsureSchema(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("store: create schema: %w", err)
	}

	return nil
}

// UpsertCourtMapping skapar eller uppdaterar en koppling mellan en
// kommunresurs och en MATCHi-bana.
func (s *Store) UpsertCourtMapping(ctx context.Context, mapping CourtMapping) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO court_mappings (kommun_resource_id, kommun_resource_name, matchi_court_id, matchi_court_name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (kommun_resource_id, matchi_court_id)
		DO UPDATE SET kommun_resource_name = EXCLUDED.kommun_resource_name,
		              matchi_court_name   = EXCLUDED.matchi_court_name,
		              updated_at          = NOW()`,
		mapping.KommunResourceID, mapping.KommunResourceName, mapping.MatchiCourtID, mapping.MatchiCourtName)
	if err != nil {
		return fmt.Errorf("store: upsert court mapping: %w", err)
	}

	return nil
}

// ListCourtMappings returnerar alla kopplingar mellan kommunresurser och
// MATCHi-banor.
func (s *Store) ListCourtMappings(ctx context.Context) ([]CourtMapping, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT kommun_resource_id, kommun_resource_name, matchi_court_id, matchi_court_name
		FROM court_mappings
		ORDER BY kommun_resource_id, matchi_court_id`)
	if err != nil {
		return nil, fmt.Errorf("store: list court mappings: %w", err)
	}
	defer rows.Close()

	mappings := []CourtMapping{}
	for rows.Next() {
		var mapping CourtMapping
		if err := rows.Scan(&mapping.KommunResourceID, &mapping.KommunResourceName, &mapping.MatchiCourtID, &mapping.MatchiCourtName); err != nil {
			return nil, fmt.Errorf("store: scan court mapping: %w", err)
		}
		mappings = append(mappings, mapping)
	}

	return mappings, rows.Err()
}

// FindBooking returnerar bokningen för en given kommunapptid och
// MATCHi-bana.
func (s *Store) FindBooking(ctx context.Context, kommunAppointmentID, matchiCourtID int64) (*Booking, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT kommun_appointment_id, kommun_resource_id, matchi_court_id, matchi_booking_id,
		       matchi_slot_id, start_time, end_time, status, created_at, updated_at
		FROM bookings
		WHERE kommun_appointment_id = $1 AND matchi_court_id = $2`,
		kommunAppointmentID, matchiCourtID)

	booking, err := scanBooking(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return booking, nil
}

// UpsertBooking skapar eller uppdaterar en bokning.
func (s *Store) UpsertBooking(ctx context.Context, booking Booking) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO bookings (kommun_appointment_id, kommun_resource_id, matchi_court_id,
		                      matchi_booking_id, matchi_slot_id, start_time, end_time, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (kommun_appointment_id, matchi_court_id)
		DO UPDATE SET kommun_resource_id = EXCLUDED.kommun_resource_id,
		              matchi_booking_id   = EXCLUDED.matchi_booking_id,
		              matchi_slot_id      = EXCLUDED.matchi_slot_id,
		              start_time          = EXCLUDED.start_time,
		              end_time            = EXCLUDED.end_time,
		              status              = EXCLUDED.status,
		              updated_at          = NOW()`,
		booking.KommunAppointmentID, booking.KommunResourceID, booking.MatchiCourtID,
		booking.MatchiBookingID, booking.MatchiSlotID, booking.StartTime, booking.EndTime, booking.Status)
	if err != nil {
		return fmt.Errorf("store: upsert booking: %w", err)
	}

	return nil
}

// ListBookingsInRange returnerar alla bokningar som startar inom perioden.
func (s *Store) ListBookingsInRange(ctx context.Context, from, to time.Time) ([]Booking, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT kommun_appointment_id, kommun_resource_id, matchi_court_id, matchi_booking_id,
		       matchi_slot_id, start_time, end_time, status, created_at, updated_at
		FROM bookings
		WHERE start_time < $2 AND end_time > $1
		ORDER BY start_time`,
		from, to)
	if err != nil {
		return nil, fmt.Errorf("store: list bookings: %w", err)
	}
	defer rows.Close()

	bookings := []Booking{}
	for rows.Next() {
		booking, err := scanBooking(rows)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, *booking)
	}

	return bookings, rows.Err()
}

// DeleteBooking tar bort en bokning ur databasen.
func (s *Store) DeleteBooking(ctx context.Context, kommunAppointmentID, matchiCourtID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM bookings
		WHERE kommun_appointment_id = $1 AND matchi_court_id = $2`,
		kommunAppointmentID, matchiCourtID)
	if err != nil {
		return fmt.Errorf("store: delete booking: %w", err)
	}

	return nil
}

// rowScanner läser en rad från en query.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanBooking läser en bokningsrad.
func scanBooking(row rowScanner) (*Booking, error) {
	var booking Booking
	if err := row.Scan(
		&booking.KommunAppointmentID,
		&booking.KommunResourceID,
		&booking.MatchiCourtID,
		&booking.MatchiBookingID,
		&booking.MatchiSlotID,
		&booking.StartTime,
		&booking.EndTime,
		&booking.Status,
		&booking.CreatedAt,
		&booking.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}

		return nil, fmt.Errorf("store: scan booking: %w", err)
	}

	return &booking, nil
}
