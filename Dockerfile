# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app

# Copy module files first for efficient caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary
RUN go build -o server server.go

# Runtime stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8888
CMD ["./server"]