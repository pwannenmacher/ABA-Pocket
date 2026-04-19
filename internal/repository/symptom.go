package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"aba-pocket/internal/models"
)

type SymptomRepository struct {
	pool *pgxpool.Pool
}

func NewSymptomRepository(pool *pgxpool.Pool) *SymptomRepository {
	return &SymptomRepository{pool: pool}
}

func (r *SymptomRepository) List(ctx context.Context) ([]*models.Symptom, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, description, source, created_at, updated_at
		FROM symptoms ORDER BY title`)
	if err != nil {
		return nil, fmt.Errorf("list symptoms: %w", err)
	}
	defer rows.Close()

	var symptoms []*models.Symptom
	for rows.Next() {
		s := &models.Symptom{}
		if err := rows.Scan(&s.ID, &s.Title, &s.Description, &s.Source, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		symptoms = append(symptoms, s)
	}
	return symptoms, rows.Err()
}

func (r *SymptomRepository) GetByID(ctx context.Context, id int64) (*models.Symptom, error) {
	s := &models.Symptom{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, title, description, source, created_at, updated_at
		FROM symptoms WHERE id = $1`, id,
	).Scan(&s.ID, &s.Title, &s.Description, &s.Source, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get symptom: %w", err)
	}

	if err := r.loadEntries(ctx, s); err != nil {
		return nil, err
	}
	if err := r.loadMedications(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (r *SymptomRepository) loadEntries(ctx context.Context, s *models.Symptom) error {
	rows, err := r.pool.Query(ctx, `
		SELECT id, left_col, right_col, sort_order
		FROM symptom_entries WHERE symptom_id = $1 ORDER BY sort_order`, s.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		e := models.CardEntry{}
		if err := rows.Scan(&e.ID, &e.LeftCol, &e.RightCol, &e.SortOrder); err != nil {
			return err
		}
		s.Entries = append(s.Entries, e)
	}
	return rows.Err()
}

func (r *SymptomRepository) loadMedications(ctx context.Context, s *models.Symptom) error {
	rows, err := r.pool.Query(ctx, `
		SELECT m.id, m.name, m.source, m.updated_at
		FROM medications m
		JOIN symptom_medications sm ON sm.medication_id = m.id
		WHERE sm.symptom_id = $1
		ORDER BY m.name`, s.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		m := &models.Medication{}
		if err := rows.Scan(&m.ID, &m.Name, &m.Source, &m.UpdatedAt); err != nil {
			return err
		}
		s.Medications = append(s.Medications, m)
	}
	return rows.Err()
}

func (r *SymptomRepository) Create(ctx context.Context, s *models.Symptom) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO symptoms (title, description, source)
		VALUES ($1, $2, $3)
		RETURNING id`,
		s.Title, s.Description, s.Source,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create symptom: %w", err)
	}
	return id, nil
}

func (r *SymptomRepository) Update(ctx context.Context, s *models.Symptom) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE symptoms SET title=$1, description=$2, source=$3, updated_at=$4
		WHERE id=$5`,
		s.Title, s.Description, s.Source, time.Now(), s.ID,
	)
	return err
}

func (r *SymptomRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM symptoms WHERE id = $1`, id)
	return err
}

func (r *SymptomRepository) ReplaceEntries(ctx context.Context, symptomID int64, entries []models.CardEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM symptom_entries WHERE symptom_id = $1`, symptomID)
	if err != nil {
		return err
	}

	for i, e := range entries {
		_, err = tx.Exec(ctx, `
			INSERT INTO symptom_entries (symptom_id, left_col, right_col, sort_order)
			VALUES ($1, $2, $3, $4)`,
			symptomID, e.LeftCol, e.RightCol, i,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *SymptomRepository) SetMedications(ctx context.Context, symptomID int64, medicationIDs []int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM symptom_medications WHERE symptom_id = $1`, symptomID)
	if err != nil {
		return err
	}

	for _, medID := range medicationIDs {
		_, err = tx.Exec(ctx, `
			INSERT INTO symptom_medications (symptom_id, medication_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`,
			symptomID, medID,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *SymptomRepository) Search(ctx context.Context, query string) ([]*models.Symptom, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, description, source, updated_at
		FROM symptoms
		WHERE to_tsvector('german', title || ' ' || COALESCE(description, '')) @@ plainto_tsquery('german', $1)
		   OR title ILIKE '%' || $1 || '%'
		ORDER BY title
		LIMIT 50`, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.Symptom
	for rows.Next() {
		s := &models.Symptom{}
		if err := rows.Scan(&s.ID, &s.Title, &s.Description, &s.Source, &s.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

func (r *SymptomRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM symptoms`).Scan(&count)
	return count, err
}

// GetLinkedMedicationIDs returns the IDs of medications linked to a symptom
func (r *SymptomRepository) GetLinkedMedicationIDs(ctx context.Context, symptomID int64) (map[int64]bool, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT medication_id FROM symptom_medications WHERE symptom_id = $1`, symptomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	return ids, rows.Err()
}
