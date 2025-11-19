# Multi-stage build for spotify-backup

# --- Build stage ---
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY spotify-backup.go ./
# Build without BuildKit cache mounts for compatibility with legacy builder
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -ldflags="-s -w" -o /out/spotify-backup ./spotify-backup.go

# --- Runtime stage ---
FROM alpine:3.20
# CA certs for HTTPS requests
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 app
USER app
WORKDIR /data

# Defaults: write outputs under /data/out
ENV OUT_DIR=/data/out
# Document the typical OAuth callback port; can be changed via SPOTIFY_REDIRECT_URI
EXPOSE 8888

COPY --from=build /out/spotify-backup /usr/local/bin/spotify-backup

# .token will be read/written relative to /data (bind-mount this from host)
ENTRYPOINT ["/usr/local/bin/spotify-backup"]

# Stage 1: Build Angular frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy package files
COPY spotify-backup-ui/package*.json ./

# Install dependencies
RUN npm ci

# Copy frontend source
COPY spotify-backup-ui/ ./

# Build Angular app for production
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.23-alpine AS backend-builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY *.go ./

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -o spotify-backup .

# Stage 3: Final runtime image
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy the Go binary from builder
COPY --from=backend-builder /app/spotify-backup .

# Copy the compiled Angular app from frontend builder
COPY --from=frontend-builder /app/frontend/dist/spotify-backup-ui/browser ./public

# Expose port
EXPOSE 8080

# Set environment variable to run in web mode
ENV WEB_MODE=true

# Run the application
CMD ["./spotify-backup"]
