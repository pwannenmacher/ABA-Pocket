package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"aba-pocket/internal/models"
)

type MedicationRepository struct {
	pool *pgxpool.Pool
}

func NewMedicationRepository(pool *pgxpool.Pool) *MedicationRepository {
	return &MedicationRepository{pool: pool}
}

func (r *MedicationRepository) List(ctx context.Context) ([]*models.Medication, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, source, created_at, updated_at
		FROM medications ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list medications: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*models.Medication, error) {
		m := &models.Medication{}
		err := row.Scan(&m.ID, &m.Name, &m.Description, &m.Source, &m.CreatedAt, &m.UpdatedAt)
		return m, err
	})
}

func (r *MedicationRepository) GetByID(ctx context.Context, id int64) (*models.Medication, error) {
	m := &models.Medication{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, source, created_at, updated_at
		FROM medications WHERE id = $1`, id,
	).Scan(&m.ID, &m.Name, &m.Description, &m.Source, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get medication: %w", err)
	}

	if err := r.loadEntries(ctx, m); err != nil {
		return nil, err
	}
	if err := r.loadSymptoms(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (r *MedicationRepository) loadEntries(ctx context.Context, m *models.Medication) error {
	rows, err := r.pool.Query(ctx, `
		SELECT id, left_col, right_col, sort_order
		FROM medication_entries WHERE medication_id = $1 ORDER BY sort_order`, m.ID)
	if err != nil {
		return err
	}
	var e models.CardEntry
	_, err = pgx.ForEachRow(rows, []any{&e.ID, &e.LeftCol, &e.RightCol, &e.SortOrder}, func() error {
		m.Entries = append(m.Entries, e)
		return nil
	})
	return err
}

func (r *MedicationRepository) loadSymptoms(ctx context.Context, m *models.Medication) error {
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.title, s.source, s.updated_at
		FROM symptoms s
		JOIN symptom_medications sm ON sm.symptom_id = s.id
		WHERE sm.medication_id = $1
		ORDER BY s.title`, m.ID)
	if err != nil {
		return err
	}
	var s models.Symptom
	_, err = pgx.ForEachRow(rows, []any{&s.ID, &s.Title, &s.Source, &s.UpdatedAt}, func() error {
		sym := s
		m.Symptoms = append(m.Symptoms, &sym)
		return nil
	})
	return err
}

func (r *MedicationRepository) Create(ctx context.Context, m *models.Medication) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO medications (name, description, source)
		VALUES ($1, $2, $3)
		RETURNING id`,
		m.Name, m.Description, m.Source,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create medication: %w", err)
	}
	return id, nil
}

func (r *MedicationRepository) Update(ctx context.Context, m *models.Medication) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE medications SET name=$1, description=$2, source=$3, updated_at=NOW()
		WHERE id=$4`,
		m.Name, m.Description, m.Source, m.ID,
	)
	return err
}

func (r *MedicationRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM medications WHERE id = $1`, id)
	return err
}

func (r *MedicationRepository) ReplaceEntries(ctx context.Context, medicationID int64, entries []models.CardEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after Commit

	if _, err := tx.Exec(ctx, `DELETE FROM medication_entries WHERE medication_id = $1`, medicationID); err != nil {
		return err
	}

	for i, e := range entries {
		if _, err := tx.Exec(ctx, `
			INSERT INTO medication_entries (medication_id, left_col, right_col, sort_order)
			VALUES ($1, $2, $3, $4)`,
			medicationID, e.LeftCol, e.RightCol, i,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *MedicationRepository) Search(ctx context.Context, query string) ([]*models.Medication, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, source, updated_at
		FROM medications
		WHERE to_tsvector('german', name || ' ' || COALESCE(description, '')) @@ plainto_tsquery('german', $1)
		   OR name ILIKE '%' || $1 || '%'
		ORDER BY name
		LIMIT 50`, query)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*models.Medication, error) {
		m := &models.Medication{}
		err := row.Scan(&m.ID, &m.Name, &m.Description, &m.Source, &m.UpdatedAt)
		return m, err
	})
}

func (r *MedicationRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM medications`).Scan(&count)
	return count, err
}
