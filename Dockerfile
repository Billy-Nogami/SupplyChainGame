FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /supply-chain-api ./cmd/api

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /supply-chain-api /usr/local/bin/supply-chain-api

EXPOSE 8080

ENV PORT=8080

ENTRYPOINT ["/usr/local/bin/supply-chain-api"]
