# Image web-hub-company — situs perusahaan HUB by Xpanca.
# Pola dua tahap sama dengan web-hub-v3: build di golang-alpine, jalan di alpine.

# ----------------------------------------------------------------------------
# Tahap 1: bangun biner Go
# ----------------------------------------------------------------------------
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO dimatikan supaya binernya statis (syarat jalan di alpine tanpa glibc).
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o web-hub-company

# ----------------------------------------------------------------------------
# Tahap 2: image jalan
# ----------------------------------------------------------------------------
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/web-hub-company .
COPY view /app/view

RUN chmod +x web-hub-company && mkdir -p /app/data

ENV PORT=3110
ENV WEB_ROOT=./view
ENV DATA_DIR=/app/data

EXPOSE 3110

CMD ["./web-hub-company"]
