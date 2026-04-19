package repository

import "github.com/jackc/pgx/v5/pgxpool"

type Repositories struct {
	Pool        *pgxpool.Pool
	Users       *UserRepository
	Symptoms    *SymptomRepository
	Medications *MedicationRepository
}

func New(pool *pgxpool.Pool) *Repositories {
	return &Repositories{
		Pool:        pool,
		Users:       NewUserRepository(pool),
		Symptoms:    NewSymptomRepository(pool),
		Medications: NewMedicationRepository(pool),
	}
}
