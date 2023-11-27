FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod  .
COPY go.sum  .
COPY main.go .

RUN go build main.go

FROM gcr.io/distroless/base

WORKDIR /

COPY --from=builder /app/main .

ENTRYPOINT ["./main"]
