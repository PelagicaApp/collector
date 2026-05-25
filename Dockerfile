FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o collector .

FROM alpine:3.21

RUN adduser -D -g '' appuser

WORKDIR /app
COPY --from=builder /app/collector .

USER appuser

EXPOSE 4000

CMD ["./collector"]