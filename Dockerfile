# syntax=docker/dockerfile:1

# Multi-stage build supporting both static and server modes

# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY . .

# Install build dependencies
RUN apk add --no-cache git

# Build both modes
RUN GOOS=js GOARCH=wasm go build -o web/main.wasm ./cmd/static && \
    cp /usr/local/go/lib/wasm/wasm_exec.js web/wasm_exec.js && \
    go build -o bin/get-server ./cmd/server

# Static mode runtime
FROM nginx:alpine AS static

COPY --from=builder /app/web /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]

# Server mode runtime
FROM alpine:3.18 AS server

RUN apk add --no-cache ca-certificates sqlite

COPY --from=builder /app/bin/get-server /usr/local/bin/get

# Create data directory
RUN mkdir -p /data && chown -R nobody:nobody /data

USER nobody

EXPOSE 8080

WORKDIR /data

CMD ["/usr/local/bin/get"]
