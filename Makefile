# ============================================================
#  EmlakPro — Makefile
#  Kullanım: make <hedef>
# ============================================================

APP     = emlakpro
BIN     = bin/$(APP)
CFG     = config.json
VERSION = $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)

.PHONY: all build run clean tidy test

## Derle (development)
build:
	@echo "Derleniyor..."
	@mkdir -p bin
	go build -ldflags="-X main.version=$(VERSION)" -o $(BIN) ./cmd/server/
	@echo "Binary: $(BIN)"

## Derle (production — küçük binary)
build-prod:
	@echo "Production build..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build -ldflags="-s -w -X main.version=$(VERSION)" \
	  -o $(BIN) ./cmd/server/
	@echo "Binary: $(BIN)"

## Çalıştır (development)
run: build
	./$(BIN) -config $(CFG)

## Bağımlılıkları indir ve temizle
tidy:
	go mod tidy
	go mod verify

## Testler
test:
	go test ./... -v

## Temizle
clean:
	rm -rf bin/

## Sunucuya deploy (ssh erişimi gerekir)
## Kullanım: make deploy HOST=root@192.168.55.45
deploy:
	@echo "Sunucuya deploy ediliyor: $(HOST)"
	ssh $(HOST) "cd /opt/emlakpro/src && git pull && bash /opt/emlakpro/deploy.sh"
