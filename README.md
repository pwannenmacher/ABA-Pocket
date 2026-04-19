# ABA Pocket

Webanwendung zur Erfassung und Generierung druckbarer **Notfallmedizin-Taschenkarten** im DIN A7-Format.

## Features

- **2 Kartentypen**: Leitsymptome & Medikamentensteckbriefe
- **Multi-Tabellen**: Leitsymptome können mehrere benannte Tabellengruppen enthalten (z.B. „Erstlinientherapie", „Bei Versagen")
- **Markdown-Inhalt**: Beide Tabellenspalten unterstützen `**fett**`, `- Listen`, Zeilenumbrüche
- **PDF-Export**: 8 DIN A7-Karten (Hochformat 74×105 mm) pro DIN A4-Seite (Querformat)
- **Einzelkarten-PDF**: Jede Karte als eigenständige DIN A7-PDF (74×105 mm, Hochformat)
- **Verlinkung**: Leitsymptome ↔ Medikamente (Many-to-Many)
- **Volltextsuche**: PostgreSQL-basiert mit HTMX Live-Dropdown
- **Smartphone-optimiert**: Mobile-first CSS, Hamburger-Menü
- **Quelle & Stand**: Auf jeder Karte sichtbar (Web + PDF-Footer)
- **Adminbereich**: CRUD für Karten, Drag & Drop-Sortierung von Tabellenzeilen
- **Benutzerverwaltung**: Admins anlegen/löschen, Session-Cookies
- **Docker**: Ein-Befehl-Deployment mit PostgreSQL

## Tech-Stack

| Schicht    | Technologie                               |
|------------|-------------------------------------------|
| Backend    | Go 1.22, `chi` Router                     |
| Datenbank  | PostgreSQL 16, `pgx/v5`                   |
| PDF        | `go-pdf/fpdf` v0.9.0                      |
| Frontend   | `html/template`, HTMX 1.9, Vanilla CSS/JS |
| Deployment | Docker + Docker Compose                   |

## Schnellstart (Docker)

```bash
# 1. Repository klonen
git clone <repo-url>
cd aba-pocket

# 2. Umgebungsvariablen anpassen
cp .env.example .env
# Pflichtfelder: SESSION_SECRET (min. 32 Zeichen), ADMIN_PASSWORD

# 3. Starten
docker compose up --build -d

# App läuft auf http://localhost:8080
# Admin: http://localhost:8080/admin
```

Der erste Admin-Account wird automatisch beim Start aus `ADMIN_USERNAME` / `ADMIN_PASSWORD` angelegt, sofern noch keine
Benutzer in der Datenbank existieren.

## Lokale Entwicklung

**Voraussetzungen**: Go 1.22+, PostgreSQL 16

```bash
# PostgreSQL starten (nur DB)
docker compose up db -d

# .env laden und Server starten
cp .env.example .env
# DATABASE_URL auf localhost anpassen
export $(grep -v '^#' .env | xargs)

go run ./cmd/server
# → http://localhost:8080
```

Für schnelle Template-Änderungen ohne Neustart: `DEV_MODE=true` in `.env` setzen (deaktiviert Template-Cache).

## Umgebungsvariablen

| Variable         | Pflicht | Standard                                                                | Beschreibung                              |
|------------------|---------|-------------------------------------------------------------------------|-------------------------------------------|
| `SESSION_SECRET` | ✅       | –                                                                       | Zufällige Zeichenkette, mind. 32 Zeichen  |
| `DATABASE_URL`   | ✅       | `postgres://aba:aba_password@localhost:5432/aba_pocket?sslmode=disable` | PostgreSQL-Verbindungs-URL                |
| `LISTEN_ADDR`    |         | `:8080`                                                                 | Bind-Adresse des HTTP-Servers             |
| `ADMIN_USERNAME` |         | –                                                                       | Benutzername für den initialen Admin      |
| `ADMIN_PASSWORD` |         | –                                                                       | Passwort für den initialen Admin          |
| `DEV_MODE`       |         | `false`                                                                 | Template-Cache deaktivieren (Entwicklung) |

## Projektstruktur

```
aba-pocket/
├── cmd/server/main.go              # Einstiegspunkt
├── internal/
│   ├── auth/auth.go                # Session-Cookie-Auth & Middleware
│   ├── config/config.go            # Konfiguration aus Env
│   ├── db/db.go                    # DB-Pool & Migration
│   ├── handlers/
│   │   ├── handlers.go             # Router, Template-Renderer, Flash-Messages
│   │   ├── public.go               # Öffentliche Seiten (Index, Karten, Suche)
│   │   ├── admin.go                # Admin-CRUD + Auth-Handler
│   │   └── pdf_handler.go          # PDF-Endpunkte
│   ├── models/models.go            # Datenstrukturen
│   ├── pdf/generator.go            # DIN A7/A4 PDF-Generierung
│   └── repository/
│       ├── repository.go           # Repositories-Aggregat
│       ├── symptom.go              # Symptome inkl. Tabellen & Volltextsuche
│       ├── medication.go           # Medikamente
│       └── user.go                 # Benutzer & Sessions
├── migrations/
│   ├── 001_initial.sql             # Basis-Schema (users, sessions, medications, symptoms)
│   └── 002_symptom_tables.sql      # Multi-Tabellen-Schema für Leitsymptome
├── web/
│   ├── templates/                  # Go html/template
│   │   ├── layout.html             # Basis-Layout (public)
│   │   ├── index.html
│   │   ├── symptom(s).html
│   │   ├── medication(s).html
│   │   ├── search.html
│   │   └── admin/
│   │       ├── layout.html         # Basis-Layout (admin)
│   │       ├── login.html
│   │       ├── dashboard.html
│   │       ├── symptoms.html
│   │       ├── symptom_form.html   # Create + Edit (Multi-Tabellen + Drag & Drop)
│   │       ├── medications.html
│   │       ├── medication_form.html
│   │       └── users.html
│   └── static/
│       ├── css/style.css           # Mobile-first, CSS-Variablen
│       └── js/app.js               # Nav-Toggle, Symptom-Tabellen-Management, Drag & Drop
├── Dockerfile                      # Multi-stage Build (builder + alpine)
├── docker-compose.yml
├── .env.example
└── Makefile
```

## PDF-Layout

### Sammel-PDF (`/pdf/all`)

```
┌──────────────────────────────────────────────────────────────────┐ DIN A4 Querformat (297×210 mm)
│ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐                            │
│ │      │ │      │ │      │ │      │  Zeile 1                    │
│ │ A7   │ │ A7   │ │ A7   │ │ A7   │  je 74×105 mm (Hochformat) │
│ │      │ │      │ │      │ │      │                             │
│ └──────┘ └──────┘ └──────┘ └──────┘                            │
│ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐                            │
│ │      │ │      │ │      │ │      │  Zeile 2                    │
│ │ A7   │ │ A7   │ │ A7   │ │ A7   │                             │
│ │      │ │      │ │      │ │      │                             │
│ └──────┘ └──────┘ └──────┘ └──────┘                            │
└──────────────────────────────────────────────────────────────────┘
  4 Spalten × 2 Zeilen = 8 Karten à 74×105 mm (DIN A7 Hochformat)
  4 × 74 mm = 296 mm + 0,5 mm Rand links/rechts = 297 mm (A4 Breite)
  2 × 105 mm = 210 mm (= A4 Höhe im Querformat, exakt)
```

### Kartenaufbau (DIN A7, 74×105 mm)

```
┌─────────────────────────────────────────┐ ← 74 mm
│ [Titelbalken, 9 mm, Farbe nach Typ]     │
│  Leitsymptom (blau) / Medikament (grün) │
├──────────────────────┬──────────────────┤
│ Medikament (fett)    │ Dosierung / Info  │ ← erste Zeile: blau hinterlegt
├──────────────────────┼──────────────────┤
│ Alternative          │ Info             │
├──────────────────────┴──────────────────┤
│  ── Tabellenüberschrift (optional) ──   │
├──────────────────────┬──────────────────┤
│ ...                  │ ...              │
├──────────────────────┴──────────────────┤
│ Quelle: ...            Stand: MM/YYYY   │ ← Footer, 5 mm, grau
└─────────────────────────────────────────┘
                                          ↑ 105 mm
```

## Datenbankschema

```sql
-- Basis
users, sessions
medications, medication_entries   -- Medikamentensteckbriefe
symptoms                          -- Leitsymptome (Metadaten)
symptom_tables                    -- Tabellengruppen eines Leitsymptoms
symptom_table_rows                -- Zeilen innerhalb einer Tabellengruppe
symptom_medications               -- Many-to-Many: Symptom ↔ Medikament

-- Volltextsuche
to_tsvector('german', title || ' ' || ...) INDEX
```

## Admin-Formular: Leitsymptome

Das Formular für Leitsymptome unterstützt mehrere dynamische Tabellengruppen:

- **Tabelle hinzufügen**: Erstellt eine neue Gruppe mit optionaler Überschrift
- **Zeile hinzufügen**: Fügt eine Zeile (linke Spalte: Medikament, rechte Spalte: Dosierung/Info) hinzu
- **Drag & Drop**: Tabellenzeilen können per Ziehgriff (⠿) umsortiert werden
- **Markdown**: Beide Spalten unterstützen `**fett**`, `- Liste`, Zeilenumbrüche
- Beim Absenden benennt JavaScript alle Felder sequenziell um (`table_N_title`, `row_N_M_med`, `row_N_M_right`)

## Makefile-Befehle

```bash
make run          # Server lokal starten
make build        # Binary nach bin/server bauen
make docker-up    # Docker Compose starten
make docker-down  # Docker Compose stoppen
make tidy         # go mod tidy
make fmt          # gofmt
```

## URL-Übersicht

| URL                         | Beschreibung                              |
|-----------------------------|-------------------------------------------|
| `GET /`                     | Startseite mit Übersicht                  |
| `GET /symptoms`             | Alle Leitsymptome                         |
| `GET /symptoms/{id}`        | Einzelne Leitsymptomkarte                 |
| `GET /medications`          | Alle Medikamente                          |
| `GET /medications/{id}`     | Einzelner Medikamentensteckbrief          |
| `GET /search?q=...`         | Suche (HTMX-fähig)                        |
| `GET /pdf/symptoms/{id}`    | Einzelkarte als PDF (DIN A7, Hochformat)  |
| `GET /pdf/medications/{id}` | Einzelkarte als PDF (DIN A7, Hochformat)  |
| `GET /pdf/all`              | Alle Karten als PDF (DIN A4, 8/Seite)     |
| `GET /admin`                | Admin-Dashboard                           |
| `GET /admin/symptoms`       | Leitsymptome verwalten                    |
| `GET /admin/medications`    | Medikamente verwalten                     |
| `GET /admin/users`          | Benutzerverwaltung                        |

## Lizenz

Privates Projekt – alle Rechte vorbehalten.
