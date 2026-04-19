# ABA Pocket

Webanwendung zur Erfassung und Generierung druckbarer **Notfallmedizin-Taschenkarten** im DIN A7-Format.

## Features

- **2 Kartentypen**: Leitsymptome & Medikamentensteckbriefe
- **Multi-Tabellen**: Leitsymptome können mehrere benannte Tabellengruppen enthalten
- **Markdown-Inhalt**: Beide Tabellenspalten unterstützen `**fett**`, `- Listen`, Zeilenumbrüche
- **PDF-Export**: 8 DIN A7-Karten (Hochformat 74×105 mm) pro DIN A4-Seite (Querformat)
- **Einzelkarten-PDF**: Jede Karte als eigenständige DIN A7-PDF (74×105 mm, Hochformat)
- **Verlinkung**: Leitsymptome ↔ Medikamente (Many-to-Many)
- **Volltextsuche**: PostgreSQL-basiert mit HTMX Live-Dropdown
- **Smartphone-optimiert**: Mobile-first CSS, Hamburger-Menü
- **Quelle & Stand**: Auf jeder Karte sichtbar (Web + PDF-Footer)
- **Adminbereich**: CRUD für Karten, Drag & Drop-Sortierung von Tabellenzeilen
- **Benutzerverwaltung**: Admins anlegen/löschen, Session-Cookies

## Tech-Stack

| Schicht    | Technologie                               |
|------------|-------------------------------------------|
| Backend    | Go 1.26, `chi` Router                     |
| Datenbank  | PostgreSQL 18, `pgx/v5`                   |
| PDF        | `go-pdf/fpdf` v0.9.0                      |
| Frontend   | `html/template`, HTMX 1.9, Vanilla CSS/JS |
| Deployment | Docker + Docker Compose                   |

[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=pwannenmacher_ABA-Pocket&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=pwannenmacher_ABA-Pocket) [![Code Smells](https://sonarcloud.io/api/project_badges/measure?project=pwannenmacher_ABA-Pocket&metric=code_smells)](https://sonarcloud.io/summary/new_code?id=pwannenmacher_ABA-Pocket) [![Coverage](https://sonarcloud.io/api/project_badges/measure?project=pwannenmacher_ABA-Pocket&metric=coverage)](https://sonarcloud.io/summary/new_code?id=pwannenmacher_ABA-Pocket) [![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=pwannenmacher_ABA-Pocket&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=pwannenmacher_ABA-Pocket) 

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

**Voraussetzungen**: Go 1.26+, PostgreSQL 18

```bash
# PostgreSQL starten (nur DB)
docker compose up db -d

# .env anlegen und anpassen
cp .env.example .env

go run .
# → http://localhost:8080
```

Die `.env`-Datei wird automatisch geladen (via `godotenv`).

Für schnelle Template-Änderungen ohne Neustart: `DEV_MODE=true` in `.env` setzen (deaktiviert Template-Cache).

## Umgebungsvariablen

| Variable         | Pflicht | Standard | Beschreibung                              |
|------------------|---------|----------|-------------------------------------------|
| `SESSION_SECRET` | ✅       | –        | Zufällige Zeichenkette, mind. 32 Zeichen  |
| `DATABASE_URL`   | ✅       | -        | PostgreSQL-Verbindungs-URL                |
| `LISTEN_ADDR`    |         | `:8080`  | Bind-Adresse des HTTP-Servers             |
| `ADMIN_USERNAME` |         | –        | Benutzername für den initialen Admin      |
| `ADMIN_PASSWORD` |         | –        | Passwort für den initialen Admin          |
| `DEV_MODE`       |         | `false`  | Template-Cache deaktivieren (Entwicklung) |
| `IMPRINT_NAME`   |         | –        | Name für Impressum (§ 5 DDG)              |
| `IMPRINT_STREET` |         | –        | Straße und Hausnummer (ggf. c/o)          |
| `IMPRINT_ZIP`    |         | –        | Postleitzahl                              |
| `IMPRINT_CITY`   |         | –        | Ort                                       |
| `IMPRINT_EMAIL`  |         | –        | Kontakt-E-Mail                            |

## PDF-Layout

### Sammel-PDF (`/pdf/all`)

```
DIN A4 Querformat (297×210 mm)
┌──────────────────────────────────────┐
│ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐  │
│ │      │ │      │ │      │ │      │  │ Zeile 1
│ │ A7   │ │ A7   │ │ A7   │ │ A7   │  │ je 74×105 mm (Hochformat)
│ │      │ │      │ │      │ │      │  │
│ └──────┘ └──────┘ └──────┘ └──────┘  │
│ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐  │
│ │      │ │      │ │      │ │      │  │ Zeile 2
│ │ A7   │ │ A7   │ │ A7   │ │ A7   │  │
│ │      │ │      │ │      │ │      │  │
│ └──────┘ └──────┘ └──────┘ └──────┘  │
└──────────────────────────────────────┘
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

| URL                         | Beschreibung                             |
|-----------------------------|------------------------------------------|
| `GET /`                     | Startseite mit Übersicht                 |
| `GET /symptoms`             | Alle Leitsymptome                        |
| `GET /symptoms/{id}`        | Einzelne Leitsymptomkarte                |
| `GET /medications`          | Alle Medikamente                         |
| `GET /medications/{id}`     | Einzelner Medikamentensteckbrief         |
| `GET /search?q=...`         | Suche (HTMX-fähig)                       |
| `GET /pdf/symptoms/{id}`    | Einzelkarte als PDF (DIN A7, Hochformat) |
| `GET /pdf/medications/{id}` | Einzelkarte als PDF (DIN A7, Hochformat) |
| `GET /pdf/all`              | Alle Karten als PDF (DIN A4, 8/Seite)    |
| `GET /admin`                | Admin-Dashboard                          |
| `GET /admin/symptoms`       | Leitsymptome verwalten                   |
| `GET /admin/medications`    | Medikamente verwalten                    |
| `GET /admin/users`          | Benutzerverwaltung                       |

## Lizenz

MIT License – siehe [LICENSE](LICENSE)
