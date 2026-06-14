FROM --platform=$BUILDPLATFORM dhi.io/golang:1 AS builder

WORKDIR /app

# Abhängigkeiten zuerst cachen – selten geändert, daher frühzeitig kopieren
COPY go.mod go.sum ./
RUN go mod download

# Nur die für den Build tatsächlich benötigten Dateien explizit kopieren.
# COPY . . würde .github/, .idea/, PROMPT.md, Makefile usw. einschließen.
COPY main.go ./
COPY internal/ internal/
COPY web/ web/
COPY migrations/ migrations/

# ARG nach COPY: COPY-Layer bleibt plattformübergreifend gecacht;
# nur der go-build-Step unterscheidet sich je Zielplattform.
# TARGETOS / TARGETARCH werden von Buildx automatisch gesetzt.
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
    go build -ldflags="-s -w" -o main .
#   -s  Symbol-Tabelle entfernen   (kein reverse engineering interner Namen)
#   -w  DWARF-Debuginfo entfernen  (kein Stack-Leak über Debug-Symbole)

FROM dhi.io/alpine-base:3.24

USER 0

WORKDIR /app

# Benutzer anlegen, bevor Dateien kopiert werden
RUN adduser -D -s /bin/sh appuser

# --chown: Eigentümer direkt beim Kopieren setzen (kein separater chown-Layer)
# --chmod=555: Nur Lesen + Ausführen erlaubt – appuser kann Dateien nicht modifizieren
COPY --chown=appuser:appuser --chmod=555 --from=builder /app/main       ./main
COPY --chown=appuser:appuser --chmod=555 --from=builder /app/web        ./web
COPY --chown=appuser:appuser --chmod=555 --from=builder /app/migrations ./migrations

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./main"]
