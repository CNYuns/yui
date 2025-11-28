.PHONY: all build clean frontend backend dev

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

all: frontend backend

frontend:
	cd frontend && npm ci && npm run build

backend:
	cd backend && CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=$(VERSION)" -o ../xpanel ./cmd/xpanel

build: frontend backend

clean:
	rm -f xpanel
	rm -rf backend/dist
	rm -rf frontend/node_modules

dev-frontend:
	cd frontend && npm run dev

dev-backend:
	cd backend && go run ./cmd/xpanel --config ../config.yaml

docker:
	docker build -t xpanel:$(VERSION) .

docker-run:
	docker-compose up -d

docker-stop:
	docker-compose down

# 交叉编译
build-linux-amd64:
	cd backend && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o ../xpanel-linux-amd64 ./cmd/xpanel

build-linux-arm64:
	cd backend && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w" -o ../xpanel-linux-arm64 ./cmd/xpanel

build-windows-amd64:
	cd backend && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o ../xpanel-windows-amd64.exe ./cmd/xpanel

build-darwin-amd64:
	cd backend && GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o ../xpanel-darwin-amd64 ./cmd/xpanel

build-darwin-arm64:
	cd backend && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w" -o ../xpanel-darwin-arm64 ./cmd/xpanel

build-all: build-linux-amd64 build-linux-arm64 build-windows-amd64 build-darwin-amd64 build-darwin-arm64
