# ABA Pocket

Webanwendung zur Erfassung und Generierung druckbarer **Notfallmedizin-Taschenkarten** im DIN A7-Format.

## Features

- **2 Kartentypen**: Leitsymptome & Medikamentensteckbriefe
- **Markdown-Inhalt**: Beide Tabellenspalten unterstützen `**fett**`, `- Listen`, Zeilenumbrüche
- **PDF-Export**: 8 DIN A7-Karten (Querformat 105×74 mm) pro DIN A4-Seite
- **Verlinkung**: Leitsymptome ↔ Medikamente (Many-to-Many)
- **Volltextsuche**: PostgreSQL-basiert mit HTMX Live-Dropdown
- **Smartphone-optimiert**: Mobile-first CSS, Hamburger-Menü
- **Quelle & Stand**: Auf jeder Karte sichtbar (Web + PDF-Footer)
- **Adminbereich**: CRUD für Karten, dynamische Tabellenzeilen per HTMX
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
│       ├── symptom.go              # Symptome (inkl. Volltextsuche)
│       ├── medication.go           # Medikamente
│       └── user.go                 # Benutzer & Sessions
├── migrations/001_initial.sql      # Datenbankschema
├── web/
│   ├── templates/                  # Go html/template
│   │   ├── layout.html             # Basis-Layout (public)
│   │   ├── index.html
│   │   ├── symptom(s).html
│   │   ├── medication(s).html
│   │   ├── search.html             # Enthält auch search_results-Block
│   │   └── admin/
│   │       ├── layout.html         # Basis-Layout (admin)
│   │       ├── login.html
│   │       ├── dashboard.html
│   │       ├── symptoms.html
│   │       ├── symptom_form.html   # Create + Edit
│   │       ├── medications.html
│   │       ├── medication_form.html
│   │       └── users.html
│   └── static/
│       ├── css/style.css           # Mobile-first, CSS-Variablen
│       └── js/app.js               # Nav-Toggle, HTMX-Config, removeRow()
├── Dockerfile                      # Multi-stage Build (builder + alpine)
├── docker-compose.yml
├── .env.example
└── Makefile
```

## PDF-Layout

```
┌─────────────────────────────────────────────────────┐ DIN A4 (210×297 mm)
│ ┌─────────────────────┐ ┌─────────────────────────┐ │
│ │ [Leitsymptom]       │ │ [Medikament]            │ │ Zeile 1
│ │ Schlüssel │ Wert    │ │ Schlüssel │ Wert        │ │
│ └─────────────────────┘ └─────────────────────────┘ │
│            ⋮                        ⋮                │
│ ┌─────────────────────┐ ┌─────────────────────────┐ │
│ │       ...           │ │         ...             │ │ Zeile 4
│ └─────────────────────┘ └─────────────────────────┘ │
└─────────────────────────────────────────────────────┘
  2 Spalten × 4 Zeilen = 8 Karten à 105×74 mm (DIN A7 quer)
```

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

| URL                         | Beschreibung                          |
|-----------------------------|---------------------------------------|
| `GET /`                     | Startseite mit Übersicht              |
| `GET /symptoms`             | Alle Leitsymptome                     |
| `GET /symptoms/{id}`        | Einzelne Leitsymptomkarte             |
| `GET /medications`          | Alle Medikamente                      |
| `GET /medications/{id}`     | Einzelner Medikamentensteckbrief      |
| `GET /search?q=...`         | Suche (HTMX-fähig)                    |
| `GET /pdf/symptoms/{id}`    | Einzelkarte als PDF (DIN A7)          |
| `GET /pdf/medications/{id}` | Einzelkarte als PDF (DIN A7)          |
| `GET /pdf/all`              | Alle Karten als PDF (DIN A4, 8/Seite) |
| `GET /admin`                | Admin-Dashboard                       |
| `GET /admin/symptoms`       | Leitsymptome verwalten                |
| `GET /admin/medications`    | Medikamente verwalten                 |
| `GET /admin/users`          | Benutzerverwaltung                    |

## Lizenz

Privates Projekt – alle Rechte vorbehalten.
