package repository

import "github.com/jackc/pgx/v5/pgxpool"

type Repositories struct {
	Users       *UserRepository
	Symptoms    *SymptomRepository
	Medications *MedicationRepository
}

func New(pool *pgxpool.Pool) *Repositories {
	return &Repositories{
		Users:       NewUserRepository(pool),
		Symptoms:    NewSymptomRepository(pool),
		Medications: NewMedicationRepository(pool),
	}
}
