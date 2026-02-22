# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder
WORKDIR /src

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/homer-server ./cmd/server

FROM alpine:3.21
RUN adduser -D -u 10001 appuser
WORKDIR /app
COPY --from=builder /out/homer-server /app/homer-server

ENV PORT=8080
EXPOSE 8080

USER appuser
ENTRYPOINT ["/app/homer-server"]
