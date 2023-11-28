FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod  .
COPY go.sum  .
COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux go build -o osmond main.go

FROM gcr.io/distroless/base

WORKDIR /

COPY --from=builder /app/osmond .

ENTRYPOINT ["./osmond"]
