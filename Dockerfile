FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.22-alpine AS backend-builder

RUN apk add --no-cache git

WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend-builder /app/backend/dist ./dist

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o /xpanel ./cmd/xpanel

FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=backend-builder /xpanel .
COPY config.example.yaml ./config.yaml

EXPOSE 8080

VOLUME ["/app/data", "/etc/xray"]

ENV TZ=Asia/Shanghai

CMD ["./xpanel", "--config", "config.yaml"]
