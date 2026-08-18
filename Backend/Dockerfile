FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/bin/server ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata wget
RUN addgroup -g 1000 app && adduser -D -u 1000 -G app app
WORKDIR /app
COPY --from=builder /app/bin/server ./server
COPY --from=builder /app/migrations ./migrations
RUN chown -R app:app /app
USER app
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 -O - http://localhost:8080/health | grep -q '"status":"ok"' || exit 1
CMD ["./server"]
