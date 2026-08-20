# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod ./

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o anchor .

# Runtime stage
FROM alpine:3.22

WORKDIR /app

COPY --from=builder /app/anchor .

EXPOSE 7379

VOLUME ["/app/data"]

CMD ["./anchor"]