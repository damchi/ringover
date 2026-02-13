FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /out/api /usr/local/bin/api
COPY --from=builder /app/pkg/translator/translation /app/pkg/translator/translation

USER app

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/api"]
