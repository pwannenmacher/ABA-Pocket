package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
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
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*models.Symptom, error) {
		s := &models.Symptom{}
		err := row.Scan(&s.ID, &s.Title, &s.Description, &s.Source, &s.CreatedAt, &s.UpdatedAt)
		return s, err
	})
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

	if err := r.loadTables(ctx, s); err != nil {
		return nil, err
	}
	if err := r.loadMedications(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (r *SymptomRepository) loadTables(ctx context.Context, s *models.Symptom) error {
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, sort_order
		FROM symptom_tables
		WHERE symptom_id = $1
		ORDER BY sort_order`, s.ID)
	if err != nil {
		return err
	}
	tbl := models.SymptomTable{SymptomID: s.ID}
	if _, err = pgx.ForEachRow(rows, []any{&tbl.ID, &tbl.Title, &tbl.SortOrder}, func() error {
		s.Tables = append(s.Tables, tbl)
		return nil
	}); err != nil {
		return err
	}
	for i := range s.Tables {
		if err := r.loadTableRows(ctx, &s.Tables[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *SymptomRepository) loadTableRows(ctx context.Context, t *models.SymptomTable) error {
	rows, err := r.pool.Query(ctx, `
		SELECT id, medication, right_col, sort_order
		FROM symptom_table_rows
		WHERE symptom_table_id = $1
		ORDER BY sort_order`, t.ID)
	if err != nil {
		return err
	}
	row := models.SymptomTableRow{SymptomTableID: t.ID}
	_, err = pgx.ForEachRow(rows, []any{&row.ID, &row.Medication, &row.RightCol, &row.SortOrder}, func() error {
		t.Rows = append(t.Rows, row)
		return nil
	})
	return err
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
	var m models.Medication
	_, err = pgx.ForEachRow(rows, []any{&m.ID, &m.Name, &m.Source, &m.UpdatedAt}, func() error {
		s.Medications = append(s.Medications, new(m))
		return nil
	})
	return err
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
		UPDATE symptoms SET title=$1, description=$2, source=$3, updated_at=NOW()
		WHERE id=$4`,
		s.Title, s.Description, s.Source, s.ID,
	)
	return err
}

func (r *SymptomRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM symptoms WHERE id = $1`, id)
	return err
}

// ReplaceTablesAndRows ersetzt alle Tabellen und Zeilen eines Leitsymptoms.
func (r *SymptomRepository) ReplaceTablesAndRows(ctx context.Context, symptomID int64, tables []models.SymptomTable) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after Commit

	// Alle bestehenden Tabellen löschen (CASCADE löscht auch Zeilen)
	if _, err := tx.Exec(ctx, `DELETE FROM symptom_tables WHERE symptom_id = $1`, symptomID); err != nil {
		return err
	}

	for i, table := range tables {
		var tableID int64
		err := tx.QueryRow(ctx, `
			INSERT INTO symptom_tables (symptom_id, title, sort_order)
			VALUES ($1, $2, $3)
			RETURNING id`,
			symptomID, table.Title, i,
		).Scan(&tableID)
		if err != nil {
			return err
		}

		for j, row := range table.Rows {
			if _, err := tx.Exec(ctx, `
				INSERT INTO symptom_table_rows (symptom_table_id, medication, right_col, sort_order)
				VALUES ($1, $2, $3, $4)`,
				tableID, row.Medication, row.RightCol, j,
			); err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}

func (r *SymptomRepository) SetMedications(ctx context.Context, symptomID int64, medicationIDs []int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after Commit

	if _, err := tx.Exec(ctx, `DELETE FROM symptom_medications WHERE symptom_id = $1`, symptomID); err != nil {
		return err
	}

	for _, medID := range medicationIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO symptom_medications (symptom_id, medication_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`,
			symptomID, medID,
		); err != nil {
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
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*models.Symptom, error) {
		s := &models.Symptom{}
		err := row.Scan(&s.ID, &s.Title, &s.Description, &s.Source, &s.UpdatedAt)
		return s, err
	})
}

func (r *SymptomRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM symptoms`).Scan(&count)
	return count, err
}

func (r *SymptomRepository) GetLinkedMedicationIDs(ctx context.Context, symptomID int64) (map[int64]bool, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT medication_id FROM symptom_medications WHERE symptom_id = $1`, symptomID)
	if err != nil {
		return nil, err
	}
	ids := make(map[int64]bool)
	var id int64
	_, err = pgx.ForEachRow(rows, []any{&id}, func() error {
		ids[id] = true
		return nil
	})
	return ids, err
}
