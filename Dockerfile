# ---------------------------------------------------------------------------
# Stage 1: Build
# ---------------------------------------------------------------------------
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w -X main.version=${VERSION}" -trimpath -o /bin/kliops-api ./cmd/kliops-api

# ---------------------------------------------------------------------------
# Stage 2: Runtime
# ---------------------------------------------------------------------------
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S kliops && adduser -S kliops -G kliops

COPY --from=builder /bin/kliops-api /usr/local/bin/kliops-api
COPY dummy_prices.xlsx /app/dummy_prices.xlsx

ENV APP_ADDR=:8070 \
    PRICING_EXCEL_PATH=/app/dummy_prices.xlsx

USER kliops

EXPOSE 8070

ENTRYPOINT ["kliops-api"]
