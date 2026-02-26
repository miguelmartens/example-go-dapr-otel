# Build stage
FROM golang:1.26.0-alpine-3.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/bin/example-go-app ./cmd/app

# Runtime stage
FROM alpine:3.23

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/bin/example-go-app .

EXPOSE 8080

ENTRYPOINT ["./example-go-app"]
