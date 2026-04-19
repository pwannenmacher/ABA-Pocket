package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"aba-pocket/internal/auth"
	"aba-pocket/internal/models"
)

// ─── Auth ──────────────────────────────────────────────────────────────────

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
	t.ExecuteTemplate(w, "login_page", map[string]interface{}{
		"Error": r.URL.Query().Get("error"),
	})
}

func (h *Handler) AdminLoginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/login?error=1", http.StatusSeeOther)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	user, err := h.repos.Users.GetByUsername(r.Context(), username)
	if err != nil || !user.IsActive {
		http.Redirect(w, r, "/admin/login?error=1", http.StatusSeeOther)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		http.Redirect(w, r, "/admin/login?error=1", http.StatusSeeOther)
		return
	}

	sessionID, err := auth.GenerateSessionID()
	if err != nil {
		http.Redirect(w, r, "/admin/login?error=1", http.StatusSeeOther)
		return
	}
	if err := h.repos.Users.CreateSession(r.Context(), sessionID, user.ID, auth.SessionDuration); err != nil {
		http.Redirect(w, r, "/admin/login?error=1", http.StatusSeeOther)
		return
	}

	auth.SetSessionCookie(w, sessionID)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *Handler) AdminLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err == nil {
		h.repos.Users.DeleteSession(r.Context(), cookie.Value)
	}
	auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

// ─── Dashboard ─────────────────────────────────────────────────────────────

func (h *Handler) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	symCount, _ := h.repos.Symptoms.Count(r.Context())
	medCount, _ := h.repos.Medications.Count(r.Context())
	userCount, _ := h.repos.Users.Count(r.Context())

	h.renderAdmin(w, http.StatusOK, "dashboard", PageData{
		Title: "Dashboard",
		User:  auth.UserFromContext(r.Context()),
		Flash: getFlash(w, r),
		Data: map[string]interface{}{
			"SymptomCount":    symCount,
			"MedicationCount": medCount,
			"UserCount":       userCount,
		},
	})
}

// ─── Symptoms ──────────────────────────────────────────────────────────────

func (h *Handler) AdminListSymptoms(w http.ResponseWriter, r *http.Request) {
	symptoms, err := h.repos.Symptoms.List(r.Context())
	if err != nil {
		http.Error(w, "Fehler", http.StatusInternalServerError)
		return
	}
	h.renderAdmin(w, http.StatusOK, "symptoms", PageData{
		Title: "Leitsymptome verwalten",
		User:  auth.UserFromContext(r.Context()),
		Flash: getFlash(w, r),
		Data:  symptoms,
	})
}

func (h *Handler) AdminNewSymptom(w http.ResponseWriter, r *http.Request) {
	medications, _ := h.repos.Medications.List(r.Context())
	h.renderAdmin(w, http.StatusOK, "symptom_form", PageData{
		Title: "Neues Leitsymptom",
		User:  auth.UserFromContext(r.Context()),
		Data: map[string]interface{}{
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
		medications, _ := h.repos.Medications.List(r.Context())
		h.renderAdmin(w, http.StatusUnprocessableEntity, "symptom_form", PageData{
			Title: "Neues Leitsymptom",
			User:  auth.UserFromContext(r.Context()),
			Data: map[string]interface{}{
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
		http.Error(w, "Fehler beim Speichern", http.StatusInternalServerError)
		return
	}

	entries := parseEntries(r)
	h.repos.Symptoms.ReplaceEntries(r.Context(), id, entries)

	medIDs := parseMedicationIDs(r)
	h.repos.Symptoms.SetMedications(r.Context(), id, medIDs)

	setFlash(w, fmt.Sprintf("Leitsymptom \"%s\" wurde erstellt.", s.Title))
	http.Redirect(w, r, "/admin/symptoms", http.StatusSeeOther)
}

func (h *Handler) AdminEditSymptom(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	symptom, err := h.repos.Symptoms.GetByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	medications, _ := h.repos.Medications.List(r.Context())
	linkedIDs, _ := h.repos.Symptoms.GetLinkedMedicationIDs(r.Context(), id)

	h.renderAdmin(w, http.StatusOK, "symptom_form", PageData{
		Title: "Leitsymptom bearbeiten",
		User:  auth.UserFromContext(r.Context()),
		Data: map[string]interface{}{
			"Symptom":           symptom,
			"AllMedications":    medications,
			"LinkedMedications": linkedIDs,
			"IsEdit":            true,
		},
	})
}

func (h *Handler) AdminUpdateSymptom(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
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
		http.Error(w, "Fehler beim Speichern", http.StatusInternalServerError)
		return
	}

	entries := parseEntries(r)
	h.repos.Symptoms.ReplaceEntries(r.Context(), id, entries)

	medIDs := parseMedicationIDs(r)
	h.repos.Symptoms.SetMedications(r.Context(), id, medIDs)

	setFlash(w, fmt.Sprintf("Leitsymptom \"%s\" wurde gespeichert.", s.Title))
	http.Redirect(w, r, "/admin/symptoms", http.StatusSeeOther)
}

func (h *Handler) AdminDeleteSymptom(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.repos.Symptoms.Delete(r.Context(), id)
	setFlash(w, "Leitsymptom wurde gelöscht.")
	http.Redirect(w, r, "/admin/symptoms", http.StatusSeeOther)
}

// ─── Medications ───────────────────────────────────────────────────────────

func (h *Handler) AdminListMedications(w http.ResponseWriter, r *http.Request) {
	medications, err := h.repos.Medications.List(r.Context())
	if err != nil {
		http.Error(w, "Fehler", http.StatusInternalServerError)
		return
	}
	h.renderAdmin(w, http.StatusOK, "medications", PageData{
		Title: "Medikamente verwalten",
		User:  auth.UserFromContext(r.Context()),
		Flash: getFlash(w, r),
		Data:  medications,
	})
}

func (h *Handler) AdminNewMedication(w http.ResponseWriter, r *http.Request) {
	h.renderAdmin(w, http.StatusOK, "medication_form", PageData{
		Title: "Neues Medikament",
		User:  auth.UserFromContext(r.Context()),
		Data: map[string]interface{}{
			"Medication": &models.Medication{},
			"IsEdit":     false,
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
		h.renderAdmin(w, http.StatusUnprocessableEntity, "medication_form", PageData{
			Title: "Neues Medikament",
			User:  auth.UserFromContext(r.Context()),
			Data: map[string]interface{}{
				"Medication": m,
				"IsEdit":     false,
				"Error":      "Bitte geben Sie einen Namen an.",
			},
		})
		return
	}

	id, err := h.repos.Medications.Create(r.Context(), m)
	if err != nil {
		http.Error(w, "Fehler beim Speichern", http.StatusInternalServerError)
		return
	}

	entries := parseEntries(r)
	h.repos.Medications.ReplaceEntries(r.Context(), id, entries)

	setFlash(w, fmt.Sprintf("Medikament \"%s\" wurde erstellt.", m.Name))
	http.Redirect(w, r, "/admin/medications", http.StatusSeeOther)
}

func (h *Handler) AdminEditMedication(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	medication, err := h.repos.Medications.GetByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	h.renderAdmin(w, http.StatusOK, "medication_form", PageData{
		Title: "Medikament bearbeiten",
		User:  auth.UserFromContext(r.Context()),
		Data: map[string]interface{}{
			"Medication": medication,
			"IsEdit":     true,
		},
	})
}

func (h *Handler) AdminUpdateMedication(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
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
		http.Error(w, "Fehler beim Speichern", http.StatusInternalServerError)
		return
	}

	entries := parseEntries(r)
	h.repos.Medications.ReplaceEntries(r.Context(), id, entries)

	setFlash(w, fmt.Sprintf("Medikament \"%s\" wurde gespeichert.", m.Name))
	http.Redirect(w, r, "/admin/medications", http.StatusSeeOther)
}

func (h *Handler) AdminDeleteMedication(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.repos.Medications.Delete(r.Context(), id)
	setFlash(w, "Medikament wurde gelöscht.")
	http.Redirect(w, r, "/admin/medications", http.StatusSeeOther)
}

// ─── Users ─────────────────────────────────────────────────────────────────

func (h *Handler) AdminListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.repos.Users.List(r.Context())
	if err != nil {
		http.Error(w, "Fehler", http.StatusInternalServerError)
		return
	}
	h.renderAdmin(w, http.StatusOK, "users", PageData{
		Title: "Benutzerverwaltung",
		User:  auth.UserFromContext(r.Context()),
		Flash: getFlash(w, r),
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
		setFlash(w, "Benutzername und Passwort sind erforderlich.")
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}
	if len(password) < 8 {
		setFlash(w, "Das Passwort muss mindestens 8 Zeichen haben.")
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
		setFlash(w, "Fehler: Benutzername existiert möglicherweise bereits.")
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}

	setFlash(w, fmt.Sprintf("Benutzer \"%s\" wurde erstellt.", username))
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (h *Handler) AdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Prevent self-deletion
	currentUser := auth.UserFromContext(r.Context())
	if currentUser != nil && currentUser.ID == id {
		setFlash(w, "Sie können Ihren eigenen Account nicht löschen.")
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}

	h.repos.Users.Delete(r.Context(), id)
	setFlash(w, "Benutzer wurde gelöscht.")
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// ─── HTMX fragments ────────────────────────────────────────────────────────

func (h *Handler) AdminEntryRow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`
<tr class="entry-row">
  <td><textarea name="entry_left[]" class="entry-input" placeholder="Schlüssel (Markdown möglich)" rows="2"></textarea></td>
  <td><textarea name="entry_right[]" class="entry-input" placeholder="Wert (Markdown möglich)" rows="2"></textarea></td>
  <td class="entry-action"><button type="button" class="btn btn-danger btn-sm" onclick="removeRow(this)">✕</button></td>
</tr>`))
}

// ─── Helpers ───────────────────────────────────────────────────────────────

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
