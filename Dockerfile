FROM golang:1.25.3-alpine AS builder

WORKDIR /src
RUN apk add --no-cache ca-certificates git tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/kliops-api ./cmd/kliops-api

FROM alpine:3.22

RUN addgroup -S kliops && adduser -S -G kliops kliops && \
    apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /out/kliops-api /app/kliops-api
COPY dummy_prices.xlsx /app/dummy_prices.xlsx

ENV APP_ADDR=:8070 \
    PRICING_EXCEL_PATH=/app/dummy_prices.xlsx

EXPOSE 8070
USER kliops
ENTRYPOINT ["/app/kliops-api"]
