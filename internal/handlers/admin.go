package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"aba-pocket/internal/auth"
	"aba-pocket/internal/models"
)

const (
	adminLoginErrorPath              = "/admin/login?error=1"
	logLoadMedicationsForSymptomForm = "load medications for symptom form: %v"
	errMsgFehlerBeimSpeichern        = "Fehler beim Speichern"
)

func (h *Handler) AdminLogin(w http.ResponseWriter, r *http.Request) {
	if auth.UserFromContext(r.Context()) != nil {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	t, err := h.getAdminTemplate("login")
	if err != nil {
		http.Error(w, "Fehler", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "login_page", map[string]any{
		"Error": r.URL.Query().Get("error"),
	}); err != nil {
		log.Printf("login template error: %v", err)
	}
}

func (h *Handler) AdminLoginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, adminLoginErrorPath, http.StatusSeeOther)
		return
	}

	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.Split(fwd, ",")[0]
	}
	if !h.loginLimiter.Allow(ip) {
		http.Redirect(w, r, "/admin/login?error=rate", http.StatusSeeOther)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	user, err := h.repos.Users.GetByUsername(r.Context(), username)
	if err != nil || !user.IsActive {
		http.Redirect(w, r, adminLoginErrorPath, http.StatusSeeOther)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		http.Redirect(w, r, adminLoginErrorPath, http.StatusSeeOther)
		return
	}

	h.loginLimiter.Reset(ip)

	sessionID, err := auth.GenerateSessionID()
	if err != nil {
		http.Redirect(w, r, adminLoginErrorPath, http.StatusSeeOther)
		return
	}
	if err := h.repos.Users.CreateSession(r.Context(), sessionID, user.ID, auth.SessionDuration); err != nil {
		http.Redirect(w, r, adminLoginErrorPath, http.StatusSeeOther)
		return
	}

	auth.SetSessionCookie(w, sessionID, !h.cfg.DevMode)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *Handler) AdminLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(auth.SessionCookieName); err == nil {
		if err := h.repos.Users.DeleteSession(r.Context(), cookie.Value); err != nil {
			log.Printf("delete session error: %v", err)
		}
	}
	auth.ClearSessionCookie(w, !h.cfg.DevMode)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	symCount, err := h.repos.Symptoms.Count(r.Context())
	if err != nil {
		log.Printf("dashboard symptom count: %v", err)
	}
	medCount, err := h.repos.Medications.Count(r.Context())
	if err != nil {
		log.Printf("dashboard medication count: %v", err)
	}
	userCount, err := h.repos.Users.Count(r.Context())
	if err != nil {
		log.Printf("dashboard user count: %v", err)
	}

	h.renderAdmin(w, r, http.StatusOK, "dashboard", PageData{
		Title: "Dashboard",
		User:  auth.UserFromContext(r.Context()),
		Flash: h.getFlash(w, r),
		Data: map[string]any{
			"SymptomCount":    symCount,
			"MedicationCount": medCount,
			"UserCount":       userCount,
		},
	})
}

func (h *Handler) AdminListSymptoms(w http.ResponseWriter, r *http.Request) {
	symptoms, err := h.repos.Symptoms.List(r.Context())
	if err != nil {
		http.Error(w, "Fehler", http.StatusInternalServerError)
		return
	}
	h.renderAdmin(w, r, http.StatusOK, "symptoms", PageData{
		Title: "Leitsymptome verwalten",
		User:  auth.UserFromContext(r.Context()),
		Flash: h.getFlash(w, r),
		Data:  symptoms,
	})
}

func (h *Handler) AdminNewSymptom(w http.ResponseWriter, r *http.Request) {
	medications, err := h.repos.Medications.List(r.Context())
	if err != nil {
		log.Printf(logLoadMedicationsForSymptomForm, err)
	}
	h.renderAdmin(w, r, http.StatusOK, "symptom_form", PageData{
		Title: "Neues Leitsymptom",
		User:  auth.UserFromContext(r.Context()),
		Data: map[string]any{
			"Symptom":           &models.Symptom{},
			"AllMedications":    medications,
			"LinkedMedications": map[int64]bool{},
			"IsEdit":            false,
		},
	})
}

func (h *Handler) AdminCreateSymptom(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Fehler", http.StatusBadRequest)
		return
	}

	s := &models.Symptom{
		Title:       strings.TrimSpace(r.FormValue("title")),
		Description: strings.TrimSpace(r.FormValue("description")),
		Source:      strings.TrimSpace(r.FormValue("source")),
	}

	if s.Title == "" {
		medications, err := h.repos.Medications.List(r.Context())
		if err != nil {
			log.Printf(logLoadMedicationsForSymptomForm, err)
		}
		h.renderAdmin(w, r, http.StatusUnprocessableEntity, "symptom_form", PageData{
			Title: "Neues Leitsymptom",
			User:  auth.UserFromContext(r.Context()),
			Data: map[string]any{
				"Symptom":           s,
				"AllMedications":    medications,
				"LinkedMedications": map[int64]bool{},
				"IsEdit":            false,
				"Error":             "Bitte geben Sie einen Titel an.",
			},
		})
		return
	}

	id, err := h.repos.Symptoms.Create(r.Context(), s)
	if err != nil {
		http.Error(w, errMsgFehlerBeimSpeichern, http.StatusInternalServerError)
		return
	}

	if !h.saveSymptomRelations(w, r, id) {
		return
	}

	h.setFlash(w, fmt.Sprintf("Leitsymptom \"%s\" wurde erstellt.", s.Title))
	http.Redirect(w, r, "/admin/symptoms", http.StatusSeeOther)
}

func (h *Handler) AdminEditSymptom(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	symptom, err := h.repos.Symptoms.GetByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	medications, err := h.repos.Medications.List(r.Context())
	if err != nil {
		log.Printf(logLoadMedicationsForSymptomForm, err)
	}
	linkedIDs, err := h.repos.Symptoms.GetLinkedMedicationIDs(r.Context(), id)
	if err != nil {
		log.Printf("load linked medication IDs: %v", err)
		linkedIDs = map[int64]bool{}
	}

	h.renderAdmin(w, r, http.StatusOK, "symptom_form", PageData{
		Title: "Leitsymptom bearbeiten",
		User:  auth.UserFromContext(r.Context()),
		Data: map[string]any{
			"Symptom":           symptom,
			"AllMedications":    medications,
			"LinkedMedications": linkedIDs,
			"IsEdit":            true,
		},
	})
}

func (h *Handler) AdminUpdateSymptom(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Fehler", http.StatusBadRequest)
		return
	}

	s := &models.Symptom{
		ID:          id,
		Title:       strings.TrimSpace(r.FormValue("title")),
		Description: strings.TrimSpace(r.FormValue("description")),
		Source:      strings.TrimSpace(r.FormValue("source")),
	}

	if err := h.repos.Symptoms.Update(r.Context(), s); err != nil {
		http.Error(w, errMsgFehlerBeimSpeichern, http.StatusInternalServerError)
		return
	}

	if !h.saveSymptomRelations(w, r, id) {
		return
	}

	h.setFlash(w, fmt.Sprintf("Leitsymptom \"%s\" wurde gespeichert.", s.Title))
	http.Redirect(w, r, "/admin/symptoms", http.StatusSeeOther)
}

func (h *Handler) AdminDeleteSymptom(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.repos.Symptoms.Delete(r.Context(), id); err != nil {
		http.Error(w, "Fehler beim Löschen", http.StatusInternalServerError)
		return
	}
	h.setFlash(w, "Leitsymptom wurde gelöscht.")
	http.Redirect(w, r, "/admin/symptoms", http.StatusSeeOther)
}

func (h *Handler) AdminListMedications(w http.ResponseWriter, r *http.Request) {
	medications, err := h.repos.Medications.List(r.Context())
	if err != nil {
		http.Error(w, "Fehler", http.StatusInternalServerError)
		return
	}
	h.renderAdmin(w, r, http.StatusOK, "medications", PageData{
		Title: "Medikamente verwalten",
		User:  auth.UserFromContext(r.Context()),
		Flash: h.getFlash(w, r),
		Data:  medications,
	})
}

func (h *Handler) AdminNewMedication(w http.ResponseWriter, r *http.Request) {
	symptoms, err := h.repos.Symptoms.List(r.Context())
	if err != nil {
		log.Printf("load symptoms for medication form: %v", err)
	}
	h.renderAdmin(w, r, http.StatusOK, "medication_form", PageData{
		Title: "Neues Medikament",
		User:  auth.UserFromContext(r.Context()),
		Data: map[string]any{
			"Medication":     &models.Medication{},
			"AllSymptoms":    symptoms,
			"LinkedSymptoms": map[int64]bool{},
			"IsEdit":         false,
		},
	})
}

func (h *Handler) AdminCreateMedication(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Fehler", http.StatusBadRequest)
		return
	}

	m := &models.Medication{
		Name:        strings.TrimSpace(r.FormValue("name")),
		Description: strings.TrimSpace(r.FormValue("description")),
		Source:      strings.TrimSpace(r.FormValue("source")),
	}

	if m.Name == "" {
		symptoms, err := h.repos.Symptoms.List(r.Context())
		if err != nil {
			log.Printf("load symptoms for medication form: %v", err)
		}
		h.renderAdmin(w, r, http.StatusUnprocessableEntity, "medication_form", PageData{
			Title: "Neues Medikament",
			User:  auth.UserFromContext(r.Context()),
			Data: map[string]any{
				"Medication":     m,
				"AllSymptoms":    symptoms,
				"LinkedSymptoms": map[int64]bool{},
				"IsEdit":         false,
				"Error":          "Bitte geben Sie einen Namen an.",
			},
		})
		return
	}

	id, err := h.repos.Medications.Create(r.Context(), m)
	if err != nil {
		http.Error(w, errMsgFehlerBeimSpeichern, http.StatusInternalServerError)
		return
	}

	entries := parseEntries(r)
	if err := h.repos.Medications.ReplaceEntries(r.Context(), id, entries); err != nil {
		http.Error(w, "Fehler beim Speichern der Einträge", http.StatusInternalServerError)
		return
	}

	symIDs := parseSymptomIDs(r)
	if err := h.repos.Medications.SetSymptoms(r.Context(), id, symIDs); err != nil {
		http.Error(w, "Fehler beim Speichern der Leitsymptome", http.StatusInternalServerError)
		return
	}

	h.setFlash(w, fmt.Sprintf("Medikament \"%s\" wurde erstellt.", m.Name))
	http.Redirect(w, r, "/admin/medications", http.StatusSeeOther)
}

func (h *Handler) AdminEditMedication(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	medication, err := h.repos.Medications.GetByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	symptoms, err := h.repos.Symptoms.List(r.Context())
	if err != nil {
		log.Printf("load symptoms for medication form: %v", err)
	}
	linkedIDs, err := h.repos.Medications.GetLinkedSymptomIDs(r.Context(), id)
	if err != nil {
		log.Printf("load linked symptom IDs: %v", err)
		linkedIDs = map[int64]bool{}
	}

	h.renderAdmin(w, r, http.StatusOK, "medication_form", PageData{
		Title: "Medikament bearbeiten",
		User:  auth.UserFromContext(r.Context()),
		Data: map[string]any{
			"Medication":     medication,
			"AllSymptoms":    symptoms,
			"LinkedSymptoms": linkedIDs,
			"IsEdit":         true,
		},
	})
}

func (h *Handler) AdminUpdateMedication(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Fehler", http.StatusBadRequest)
		return
	}

	m := &models.Medication{
		ID:          id,
		Name:        strings.TrimSpace(r.FormValue("name")),
		Description: strings.TrimSpace(r.FormValue("description")),
		Source:      strings.TrimSpace(r.FormValue("source")),
	}

	if err := h.repos.Medications.Update(r.Context(), m); err != nil {
		http.Error(w, errMsgFehlerBeimSpeichern, http.StatusInternalServerError)
		return
	}

	entries := parseEntries(r)
	if err := h.repos.Medications.ReplaceEntries(r.Context(), id, entries); err != nil {
		http.Error(w, "Fehler beim Speichern der Einträge", http.StatusInternalServerError)
		return
	}

	symIDs := parseSymptomIDs(r)
	if err := h.repos.Medications.SetSymptoms(r.Context(), id, symIDs); err != nil {
		http.Error(w, "Fehler beim Speichern der Leitsymptome", http.StatusInternalServerError)
		return
	}

	h.setFlash(w, fmt.Sprintf("Medikament \"%s\" wurde gespeichert.", m.Name))
	http.Redirect(w, r, "/admin/medications", http.StatusSeeOther)
}

func (h *Handler) AdminDeleteMedication(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.repos.Medications.Delete(r.Context(), id); err != nil {
		http.Error(w, "Fehler beim Löschen", http.StatusInternalServerError)
		return
	}
	h.setFlash(w, "Medikament wurde gelöscht.")
	http.Redirect(w, r, "/admin/medications", http.StatusSeeOther)
}

func (h *Handler) AdminListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.repos.Users.List(r.Context())
	if err != nil {
		http.Error(w, "Fehler", http.StatusInternalServerError)
		return
	}
	h.renderAdmin(w, r, http.StatusOK, "users", PageData{
		Title: "Benutzerverwaltung",
		User:  auth.UserFromContext(r.Context()),
		Flash: h.getFlash(w, r),
		Data:  users,
	})
}

func (h *Handler) AdminCreateUser(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Fehler", http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	if username == "" || password == "" {
		h.setFlash(w, "Benutzername und Passwort sind erforderlich.")
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}
	if len(password) < 8 {
		h.setFlash(w, "Das Passwort muss mindestens 8 Zeichen haben.")
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Fehler", http.StatusInternalServerError)
		return
	}

	_, err = h.repos.Users.Create(r.Context(), username, email, string(hash))
	if err != nil {
		h.setFlash(w, "Fehler: Benutzername existiert möglicherweise bereits.")
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}

	h.setFlash(w, fmt.Sprintf("Benutzer \"%s\" wurde erstellt.", username))
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (h *Handler) AdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Prevent self-deletion
	currentUser := auth.UserFromContext(r.Context())
	if currentUser != nil && currentUser.ID == id {
		h.setFlash(w, "Sie können Ihren eigenen Account nicht löschen.")
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}

	if err := h.repos.Users.Delete(r.Context(), id); err != nil {
		http.Error(w, "Fehler beim Löschen", http.StatusInternalServerError)
		return
	}
	h.setFlash(w, "Benutzer wurde gelöscht.")
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// parseID parses the "id" URL parameter as int64.
func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

// saveSymptomRelations speichert Tabellen und Medikamentenverknüpfungen eines Leitsymptoms.
func (h *Handler) saveSymptomRelations(w http.ResponseWriter, r *http.Request, id int64) bool {
	tables := parseSymptomTables(r)
	if err := h.repos.Symptoms.ReplaceTablesAndRows(r.Context(), id, tables); err != nil {
		http.Error(w, "Fehler beim Speichern der Tabellen", http.StatusInternalServerError)
		return false
	}
	medIDs := parseMedicationIDs(r)
	if err := h.repos.Symptoms.SetMedications(r.Context(), id, medIDs); err != nil {
		http.Error(w, "Fehler beim Speichern der Medikamente", http.StatusInternalServerError)
		return false
	}
	return true
}

// parseSymptomTables liest die Tabellen-Struktur aus dem Formular.
// Das JS renumberSymptomTables() benennt die Felder vor dem Submit um:
//
//	table_N_title, row_count_N, row_N_M_med, row_N_M_right
func parseSymptomTables(r *http.Request) []models.SymptomTable {
	tableCount, _ := strconv.Atoi(r.FormValue("table_count"))
	var tables []models.SymptomTable

	for t := 0; t < tableCount; t++ {
		title := strings.TrimSpace(r.FormValue(fmt.Sprintf("table_%d_title", t)))
		rowCount, _ := strconv.Atoi(r.FormValue(fmt.Sprintf("row_count_%d", t)))

		var rows []models.SymptomTableRow
		for m := 0; m < rowCount; m++ {
			med := strings.TrimSpace(r.FormValue(fmt.Sprintf("row_%d_%d_med", t, m)))
			right := strings.TrimSpace(r.FormValue(fmt.Sprintf("row_%d_%d_right", t, m)))
			if med == "" && right == "" {
				continue
			}
			rows = append(rows, models.SymptomTableRow{
				Medication: med,
				RightCol:   right,
				SortOrder:  m,
			})
		}

		if len(rows) > 0 || title != "" {
			tables = append(tables, models.SymptomTable{
				Title:     title,
				Rows:      rows,
				SortOrder: t,
			})
		}
	}
	return tables
}

func parseEntries(r *http.Request) []models.CardEntry {
	leftCols := r.Form["entry_left[]"]
	rightCols := r.Form["entry_right[]"]

	var entries []models.CardEntry
	for i, left := range leftCols {
		right := ""
		if i < len(rightCols) {
			right = rightCols[i]
		}
		left = strings.TrimSpace(left)
		right = strings.TrimSpace(right)
		if left == "" && right == "" {
			continue // skip empty rows
		}
		entries = append(entries, models.CardEntry{
			LeftCol:   left,
			RightCol:  right,
			SortOrder: i,
		})
	}
	return entries
}

func parseMedicationIDs(r *http.Request) []int64 {
	raw := r.Form["medication_ids[]"]
	ids := make([]int64, 0, len(raw))
	for _, s := range raw {
		id, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func parseSymptomIDs(r *http.Request) []int64 {
	raw := r.Form["symptom_ids[]"]
	ids := make([]int64, 0, len(raw))
	for _, s := range raw {
		id, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}
