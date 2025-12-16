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

ARG VERSION=1.4.0
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w -X main.Version=${VERSION}" -o /y-ui-server ./cmd/y-ui

# 使用固定版本的 Alpine 以确保安全性和可重现性
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# 创建非 root 用户运行服务
RUN adduser -D -u 1000 yui

WORKDIR /usr/local/y-ui

COPY --from=backend-builder /y-ui-server .
COPY config.example.yaml ./config.yaml

RUN mkdir -p /var/log/y-ui /etc/xray && \
    chown -R yui:yui /usr/local/y-ui /var/log/y-ui /etc/xray && \
    chmod 600 config.yaml

EXPOSE 8080

VOLUME ["/usr/local/y-ui", "/etc/xray", "/var/log/y-ui"]

ENV TZ=Asia/Shanghai
ENV GIN_MODE=release

# 切换到非 root 用户
USER yui

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/system/status || exit 1

CMD ["./y-ui-server", "--config", "config.yaml"]
