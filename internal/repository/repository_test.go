package repository

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"aba-pocket/internal/models"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	pgC, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("test_aba"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		panic("start postgres container: " + err.Error())
	}

	connStr, err := pgC.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("get connection string: " + err.Error())
	}

	testPool, err = pgxpool.New(ctx, connStr)
	if err != nil {
		panic("connect to test db: " + err.Error())
	}

	// Run migration
	migrationPath := filepath.Join("..", "..", "migrations", "001_schema.sql")
	data, err := os.ReadFile(migrationPath)
	if err != nil {
		panic("read migration: " + err.Error())
	}
	if _, err := testPool.Exec(ctx, string(data)); err != nil {
		panic("run migration: " + err.Error())
	}

	code := m.Run()

	testPool.Close()
	_ = pgC.Terminate(ctx)
	os.Exit(code)
}

func cleanDB(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	tables := []string{"symptom_medications", "symptom_table_rows", "symptom_tables", "medication_entries", "sessions", "symptoms", "medications", "users"}
	for _, tbl := range tables {
		if _, err := testPool.Exec(ctx, "DELETE FROM "+tbl); err != nil {
			t.Fatalf("clean %s: %v", tbl, err)
		}
	}
}

// ── User Repository ────────────────────────────────────────────────────────

func TestUserRepository_CreateAndGet(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewUserRepository(testPool)

	u, err := repo.Create(ctx, "admin", "a@b.de", "$2a$10$hashhashhash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if u.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if u.Username != "admin" {
		t.Errorf("username = %q", u.Username)
	}
	if !u.IsActive {
		t.Error("new user should be active")
	}

	got, err := repo.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Username != "admin" {
		t.Errorf("GetByID username = %q", got.Username)
	}

	got2, err := repo.GetByUsername(ctx, "admin")
	if err != nil {
		t.Fatalf("GetByUsername: %v", err)
	}
	if got2.ID != u.ID {
		t.Errorf("GetByUsername ID mismatch")
	}
}

func TestUserRepository_List(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewUserRepository(testPool)

	repo.Create(ctx, "bob", "", "hash1")
	repo.Create(ctx, "alice", "", "hash2")

	users, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].Username != "alice" {
		t.Error("should be sorted by username")
	}
}

func TestUserRepository_Delete(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewUserRepository(testPool)

	u, _ := repo.Create(ctx, "todelete", "", "hash")
	if err := repo.Delete(ctx, u.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	count, _ := repo.Count(ctx)
	if count != 0 {
		t.Errorf("expected 0 users after delete, got %d", count)
	}
}

func TestUserRepository_Count(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewUserRepository(testPool)

	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	repo.Create(ctx, "u1", "", "h")
	count, _ = repo.Count(ctx)
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

func TestUserRepository_Sessions(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewUserRepository(testPool)

	u, _ := repo.Create(ctx, "sessuser", "", "hash")

	err := repo.CreateSession(ctx, "sess-abc-123", u.ID, 24*time.Hour)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	s, err := repo.GetSession(ctx, "sess-abc-123")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if s.UserID != u.ID {
		t.Errorf("session user ID = %d, want %d", s.UserID, u.ID)
	}

	// Expired session should not be found
	err = repo.CreateSession(ctx, "sess-expired", u.ID, -1*time.Hour)
	if err != nil {
		t.Fatalf("create expired session: %v", err)
	}
	_, err = repo.GetSession(ctx, "sess-expired")
	if err == nil {
		t.Error("expired session should not be returned")
	}

	// Delete session
	if err := repo.DeleteSession(ctx, "sess-abc-123"); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	_, err = repo.GetSession(ctx, "sess-abc-123")
	if err == nil {
		t.Error("deleted session should not be found")
	}
}

func TestUserRepository_DeleteExpiredSessions(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewUserRepository(testPool)

	u, _ := repo.Create(ctx, "user", "", "hash")
	repo.CreateSession(ctx, "valid", u.ID, 24*time.Hour)
	repo.CreateSession(ctx, "expired1", u.ID, -1*time.Hour)
	repo.CreateSession(ctx, "expired2", u.ID, -2*time.Hour)

	n, err := repo.DeleteExpiredSessions(ctx)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 deleted, got %d", n)
	}

	// Valid session should still exist
	_, err = repo.GetSession(ctx, "valid")
	if err != nil {
		t.Error("valid session should still exist")
	}
}

// ── Symptom Repository ─────────────────────────────────────────────────────

func TestSymptomRepository_CRUD(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewSymptomRepository(testPool)

	s := &models.Symptom{Title: "Anaphylaxie", Description: "Akut", Source: "AWMF"}
	id, err := repo.Create(ctx, s)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero ID")
	}

	got, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != "Anaphylaxie" {
		t.Errorf("title = %q", got.Title)
	}

	got.Title = "Anaphylaxie (Update)"
	got.ID = id
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}
	got2, _ := repo.GetByID(ctx, id)
	if got2.Title != "Anaphylaxie (Update)" {
		t.Errorf("updated title = %q", got2.Title)
	}

	if err := repo.Delete(ctx, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err = repo.GetByID(ctx, id)
	if err == nil {
		t.Error("should not find deleted symptom")
	}
}

func TestSymptomRepository_ListAndCount(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewSymptomRepository(testPool)

	repo.Create(ctx, &models.Symptom{Title: "B-Symptom"})
	repo.Create(ctx, &models.Symptom{Title: "A-Symptom"})

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2, got %d", len(list))
	}
	if list[0].Title != "A-Symptom" {
		t.Error("should be sorted by title")
	}

	count, _ := repo.Count(ctx)
	if count != 2 {
		t.Errorf("count = %d", count)
	}
}

func TestSymptomRepository_ReplaceTablesAndRows(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewSymptomRepository(testPool)

	id, _ := repo.Create(ctx, &models.Symptom{Title: "Test"})

	tables := []models.SymptomTable{
		{
			Title: "Erstlinie",
			Rows: []models.SymptomTableRow{
				{Medication: "Adrenalin", RightCol: "0,5 mg"},
				{Medication: "Volumen", RightCol: "500 ml"},
			},
		},
		{
			Title: "Zweitlinie",
			Rows: []models.SymptomTableRow{
				{Medication: "Prednisolon", RightCol: "250 mg"},
			},
		},
	}

	if err := repo.ReplaceTablesAndRows(ctx, id, tables); err != nil {
		t.Fatalf("replace: %v", err)
	}

	got, _ := repo.GetByID(ctx, id)
	if len(got.Tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(got.Tables))
	}
	if got.Tables[0].Title != "Erstlinie" {
		t.Errorf("table 0 title = %q", got.Tables[0].Title)
	}
	if len(got.Tables[0].Rows) != 2 {
		t.Errorf("table 0 rows = %d", len(got.Tables[0].Rows))
	}
	if got.Tables[1].Rows[0].Medication != "Prednisolon" {
		t.Errorf("table 1 row 0 med = %q", got.Tables[1].Rows[0].Medication)
	}

	// Replace with fewer tables
	if err := repo.ReplaceTablesAndRows(ctx, id, tables[:1]); err != nil {
		t.Fatalf("replace again: %v", err)
	}
	got2, _ := repo.GetByID(ctx, id)
	if len(got2.Tables) != 1 {
		t.Errorf("after replace: expected 1 table, got %d", len(got2.Tables))
	}
}

func TestSymptomRepository_Search(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewSymptomRepository(testPool)

	repo.Create(ctx, &models.Symptom{Title: "Anaphylaxie"})
	repo.Create(ctx, &models.Symptom{Title: "Reanimation"})

	results, err := repo.Search(ctx, "Anaphylaxie")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	// ILIKE fallback
	results2, _ := repo.Search(ctx, "reanim")
	if len(results2) != 1 {
		t.Errorf("ILIKE search: expected 1 result, got %d", len(results2))
	}
}

func TestSymptomRepository_SetMedications(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	symRepo := NewSymptomRepository(testPool)
	medRepo := NewMedicationRepository(testPool)

	symID, _ := symRepo.Create(ctx, &models.Symptom{Title: "Test"})
	medID1, _ := medRepo.Create(ctx, &models.Medication{Name: "Med1"})
	medID2, _ := medRepo.Create(ctx, &models.Medication{Name: "Med2"})

	if err := symRepo.SetMedications(ctx, symID, []int64{medID1, medID2}); err != nil {
		t.Fatalf("set medications: %v", err)
	}

	linked, err := symRepo.GetLinkedMedicationIDs(ctx, symID)
	if err != nil {
		t.Fatalf("get linked: %v", err)
	}
	if len(linked) != 2 {
		t.Errorf("expected 2 linked, got %d", len(linked))
	}
	if !linked[medID1] || !linked[medID2] {
		t.Error("both meds should be linked")
	}

	// Verify loaded via GetByID
	sym, _ := symRepo.GetByID(ctx, symID)
	if len(sym.Medications) != 2 {
		t.Errorf("expected 2 medications on symptom, got %d", len(sym.Medications))
	}

	// Replace with subset
	symRepo.SetMedications(ctx, symID, []int64{medID1})
	linked2, _ := symRepo.GetLinkedMedicationIDs(ctx, symID)
	if len(linked2) != 1 {
		t.Errorf("after replace: expected 1 linked, got %d", len(linked2))
	}
}

// ── Medication Repository ──────────────────────────────────────────────────

func TestMedicationRepository_CRUD(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewMedicationRepository(testPool)

	m := &models.Medication{Name: "Adrenalin", Description: "Katecholamin", Source: "Rote Liste"}
	id, err := repo.Create(ctx, m)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "Adrenalin" {
		t.Errorf("name = %q", got.Name)
	}

	got.Name = "Adrenalin (Epinephrin)"
	got.ID = id
	repo.Update(ctx, got)
	got2, _ := repo.GetByID(ctx, id)
	if got2.Name != "Adrenalin (Epinephrin)" {
		t.Errorf("updated name = %q", got2.Name)
	}

	repo.Delete(ctx, id)
	_, err = repo.GetByID(ctx, id)
	if err == nil {
		t.Error("should not find deleted medication")
	}
}

func TestMedicationRepository_ListAndCount(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewMedicationRepository(testPool)

	repo.Create(ctx, &models.Medication{Name: "Zzz"})
	repo.Create(ctx, &models.Medication{Name: "Aaa"})

	list, _ := repo.List(ctx)
	if len(list) != 2 {
		t.Fatalf("expected 2, got %d", len(list))
	}
	if list[0].Name != "Aaa" {
		t.Error("should be sorted by name")
	}

	count, _ := repo.Count(ctx)
	if count != 2 {
		t.Errorf("count = %d", count)
	}
}

func TestMedicationRepository_ReplaceEntries(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewMedicationRepository(testPool)

	id, _ := repo.Create(ctx, &models.Medication{Name: "Test"})

	entries := []models.CardEntry{
		{LeftCol: "Wirkstoff", RightCol: "Epinephrin"},
		{LeftCol: "Dosierung", RightCol: "0,5 mg"},
	}

	if err := repo.ReplaceEntries(ctx, id, entries); err != nil {
		t.Fatalf("replace: %v", err)
	}

	got, _ := repo.GetByID(ctx, id)
	if len(got.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got.Entries))
	}
	if got.Entries[0].LeftCol != "Wirkstoff" {
		t.Errorf("entry 0 left = %q", got.Entries[0].LeftCol)
	}
	if got.Entries[1].RightCol != "0,5 mg" {
		t.Errorf("entry 1 right = %q", got.Entries[1].RightCol)
	}

	// Replace with different entries
	repo.ReplaceEntries(ctx, id, entries[:1])
	got2, _ := repo.GetByID(ctx, id)
	if len(got2.Entries) != 1 {
		t.Errorf("after replace: expected 1 entry, got %d", len(got2.Entries))
	}
}

func TestMedicationRepository_Search(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewMedicationRepository(testPool)

	repo.Create(ctx, &models.Medication{Name: "Adrenalin"})
	repo.Create(ctx, &models.Medication{Name: "Atropin"})

	results, _ := repo.Search(ctx, "Adrenalin")
	if len(results) != 1 {
		t.Errorf("expected 1, got %d", len(results))
	}

	results2, _ := repo.Search(ctx, "atrop")
	if len(results2) != 1 {
		t.Errorf("ILIKE: expected 1, got %d", len(results2))
	}
}

func TestMedicationRepository_LinkedSymptoms(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	symRepo := NewSymptomRepository(testPool)
	medRepo := NewMedicationRepository(testPool)

	medID, _ := medRepo.Create(ctx, &models.Medication{Name: "Adrenalin"})
	symID, _ := symRepo.Create(ctx, &models.Symptom{Title: "Anaphylaxie"})
	symRepo.SetMedications(ctx, symID, []int64{medID})

	med, _ := medRepo.GetByID(ctx, medID)
	if len(med.Symptoms) != 1 {
		t.Fatalf("expected 1 linked symptom, got %d", len(med.Symptoms))
	}
	if med.Symptoms[0].Title != "Anaphylaxie" {
		t.Errorf("linked symptom title = %q", med.Symptoms[0].Title)
	}
}

// ── Cascade Delete ─────────────────────────────────────────────────────────

func TestCascadeDelete_SymptomDeletesCascade(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	symRepo := NewSymptomRepository(testPool)
	medRepo := NewMedicationRepository(testPool)

	symID, _ := symRepo.Create(ctx, &models.Symptom{Title: "Test"})
	medID, _ := medRepo.Create(ctx, &models.Medication{Name: "Med"})

	symRepo.ReplaceTablesAndRows(ctx, symID, []models.SymptomTable{
		{Title: "T1", Rows: []models.SymptomTableRow{{Medication: "A", RightCol: "B"}}},
	})
	symRepo.SetMedications(ctx, symID, []int64{medID})

	symRepo.Delete(ctx, symID)

	// Tables, rows, and links should be gone
	var count int
	testPool.QueryRow(ctx, "SELECT COUNT(*) FROM symptom_tables WHERE symptom_id = $1", symID).Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 tables after cascade delete, got %d", count)
	}
	testPool.QueryRow(ctx, "SELECT COUNT(*) FROM symptom_medications WHERE symptom_id = $1", symID).Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 medication links after cascade delete, got %d", count)
	}
}

func TestCascadeDelete_UserDeletesSessions(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewUserRepository(testPool)

	u, _ := repo.Create(ctx, "user", "", "hash")
	repo.CreateSession(ctx, "sess1", u.ID, 24*time.Hour)

	repo.Delete(ctx, u.ID)

	var count int
	testPool.QueryRow(ctx, "SELECT COUNT(*) FROM sessions WHERE user_id = $1", u.ID).Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 sessions after user delete, got %d", count)
	}
}

// ── Sort order preserved ───────────────────────────────────────────────────

func TestSortOrderPreserved(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	symRepo := NewSymptomRepository(testPool)

	id, _ := symRepo.Create(ctx, &models.Symptom{Title: "Order Test"})
	tables := []models.SymptomTable{
		{Title: "Second", Rows: []models.SymptomTableRow{{Medication: "B"}}},
		{Title: "First", Rows: []models.SymptomTableRow{{Medication: "A"}}},
	}
	symRepo.ReplaceTablesAndRows(ctx, id, tables)

	got, _ := symRepo.GetByID(ctx, id)
	titles := make([]string, len(got.Tables))
	for i, tbl := range got.Tables {
		titles[i] = tbl.Title
	}
	if !sort.StringsAreSorted([]string{"Second", "First"}) {
		// They should come back in insertion order (sort_order 0, 1)
		if titles[0] != "Second" || titles[1] != "First" {
			t.Errorf("table order = %v, want [Second, First]", titles)
		}
	}
}
