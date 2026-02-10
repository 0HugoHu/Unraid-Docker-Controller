# Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Build backend
FROM golang:1.24-alpine AS backend
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY go.mod ./

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Copy built frontend into the expected location
COPY --from=frontend /app/internal/api/static ./internal/api/static

# Download dependencies and build the binary
RUN go mod tidy && \
    CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o controller ./cmd/controller

# Final image
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache git docker-cli ca-certificates tzdata

WORKDIR /app

# Copy the binary
COPY --from=backend /app/controller .

# Create data directory
RUN mkdir -p /data

EXPOSE 13000

ENTRYPOINT ["/app/controller"]
