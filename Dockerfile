FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend
COPY frontend/package.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

FROM golang:1.22-alpine AS backend-builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /app/backend
COPY backend/go.mod ./
RUN go mod download || true
COPY backend/ ./
RUN go mod tidy
COPY --from=frontend-builder /app/frontend/dist ./dist

ARG VERSION=1.1.8
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w -X main.Version=${VERSION}" -o /y-ui-server ./cmd/y-ui

FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /usr/local/y-ui

COPY --from=backend-builder /y-ui-server .
COPY config.example.yaml ./config.yaml

RUN mkdir -p /var/log/y-ui /etc/xray

EXPOSE 8080

VOLUME ["/usr/local/y-ui", "/etc/xray", "/var/log/y-ui"]

ENV TZ=Asia/Shanghai
ENV GIN_MODE=release

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/system/status || exit 1

CMD ["./y-ui-server", "--config", "config.yaml"]
