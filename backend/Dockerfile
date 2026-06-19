# Build stage
FROM golang:1.22-alpine AS builder

# Set working directory
WORKDIR /app

# Copy dependency files
COPY go.mod ./
# Copy go.sum if it exists (for now it doesn't, we'll download when files copy or mod tidy runs)
RUN if [ -f go.sum ]; then COPY go.sum ./; fi

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the statically linked Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/server/main.go

# Production stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

# Create non-root user for security
RUN adduser -D -u 10001 appuser
USER appuser

WORKDIR /home/appuser

# Copy compile binary from build stage
COPY --from=builder /app/server .

# Expose port
EXPOSE 8080

# Run the app
ENTRYPOINT ["./server"]
