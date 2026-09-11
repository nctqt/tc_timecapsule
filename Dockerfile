# Stage 1: Build the binary
FROM golang:1.26-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build from the correct nested path
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main ./cmd/api

# Stage 2: Run the binary in a tiny production container
FROM alpine:latest
WORKDIR /root/

COPY --from=builder /app/main .
RUN chmod +x ./main

EXPOSE 8080
CMD ["./main"]
