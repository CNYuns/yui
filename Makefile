.PHONY: all build clean frontend backend dev install uninstall

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "1.1.2")
BINARY_NAME = y-ui-server

all: frontend backend

frontend:
	cd frontend && npm ci && npm run build

backend:
	cd backend && CGO_ENABLED=0 go build -ldflags "-s -w -X main.Version=$(VERSION)" -o ../$(BINARY_NAME) ./cmd/y-ui

build: frontend backend

clean:
	rm -f $(BINARY_NAME) y-ui-*
	rm -rf backend/dist
	rm -rf frontend/node_modules

dev-frontend:
	cd frontend && npm run dev

dev-backend:
	cd backend && go run ./cmd/y-ui --config ../config.yaml

docker:
	docker build -t y-ui:$(VERSION) .

docker-run:
	docker-compose up -d

docker-stop:
	docker-compose down

# 交叉编译
build-linux-amd64:
	cd backend && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X main.Version=$(VERSION)" -o ../y-ui-linux-amd64 ./cmd/y-ui

build-linux-arm64:
	cd backend && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w -X main.Version=$(VERSION)" -o ../y-ui-linux-arm64 ./cmd/y-ui

build-linux-arm:
	cd backend && GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -ldflags "-s -w -X main.Version=$(VERSION)" -o ../y-ui-linux-arm ./cmd/y-ui

build-windows-amd64:
	cd backend && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X main.Version=$(VERSION)" -o ../y-ui-windows-amd64.exe ./cmd/y-ui

build-windows-arm64:
	cd backend && GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w -X main.Version=$(VERSION)" -o ../y-ui-windows-arm64.exe ./cmd/y-ui

build-darwin-amd64:
	cd backend && GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X main.Version=$(VERSION)" -o ../y-ui-darwin-amd64 ./cmd/y-ui

build-darwin-arm64:
	cd backend && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w -X main.Version=$(VERSION)" -o ../y-ui-darwin-arm64 ./cmd/y-ui

build-all: frontend build-linux-amd64 build-linux-arm64 build-linux-arm build-windows-amd64 build-windows-arm64 build-darwin-amd64 build-darwin-arm64

# 本地安装
install: build
	@echo "安装 Y-UI..."
	mkdir -p /usr/local/y-ui
	mkdir -p /var/log/y-ui
	mkdir -p /etc/xray
	cp $(BINARY_NAME) /usr/local/y-ui/
	cp -r backend/dist /usr/local/y-ui/ 2>/dev/null || true
	test -f /usr/local/y-ui/config.yaml || cp config.example.yaml /usr/local/y-ui/config.yaml
	cp y-ui.service /etc/systemd/system/
	systemctl daemon-reload
	systemctl enable y-ui
	@echo "安装完成! 使用 'systemctl start y-ui' 启动服务"

uninstall:
	@echo "卸载 Y-UI..."
	systemctl stop y-ui 2>/dev/null || true
	systemctl disable y-ui 2>/dev/null || true
	rm -f /etc/systemd/system/y-ui.service
	rm -rf /usr/local/y-ui
	rm -f /usr/local/bin/y-ui
	rm -rf /var/log/y-ui
	systemctl daemon-reload
	@echo "卸载完成"
