FROM --platform=$BUILDPLATFORM dhi.io/golang:1 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# TARGETOS / TARGETARCH werden von Buildx automatisch gesetzt
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -a -installsuffix cgo -o main .

FROM dhi.io/alpine-base:3.24

USER 0

WORKDIR /app

COPY --from=builder /app/main /app/main
COPY --from=builder /app/web /app/web
COPY --from=builder /app/migrations /app/migrations

RUN adduser -D -s /bin/sh appuser && chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./main"]
