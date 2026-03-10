.PHONY: build build-web build-local build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64 docker-build docker-build-cross docker-build-cross-push docker-build-linux-amd64 docker-build-linux-arm64 test run

build-web:
	cd web && npm ci && npm run build
	rm -rf cmd/server/static
	cp -r web/dist cmd/server/static

# 各平台单独打包，产物输出到 dist/
build-linux-amd64: build-web
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/rss-reader-linux-amd64 ./cmd/server

build-linux-arm64: build-web
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o dist/rss-reader-linux-arm64 ./cmd/server

build-darwin-amd64: build-web
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o dist/rss-reader-darwin-amd64 ./cmd/server

build-darwin-arm64: build-web
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o dist/rss-reader-darwin-arm64 ./cmd/server

build-windows-amd64: build-web
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/rss-reader-windows-amd64.exe ./cmd/server

# 打包全部平台（build-web 只执行一次）
build: build-web
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o dist/rss-reader-linux-amd64     ./cmd/server
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -o dist/rss-reader-linux-arm64     ./cmd/server
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -o dist/rss-reader-darwin-amd64    ./cmd/server
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o dist/rss-reader-darwin-arm64    ./cmd/server
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/rss-reader-windows-amd64.exe ./cmd/server
	@echo "多平台构建完成，产物在 dist/ 目录"

# 仅当前平台单二进制（便于本地 ./rss-reader 运行）
build-local: build-web
	CGO_ENABLED=0 go build -o rss-reader ./cmd/server

# run 前会先构建前端并拷贝到 cmd/server/static，保证用最新前端
run: build-web
	go run ./cmd/server

test:
	go test ./...

docker-build:
	docker build -t rss-reader:latest .

# 跨平台 Docker 构建，输出 tar 到 dist/ (需先执行: docker buildx create --use)
docker-build-cross:
	@mkdir -p dist
	docker buildx build --platform linux/amd64 -o type=docker,dest=dist/rss-reader-linux-amd64.tar .
	docker buildx build --platform linux/arm64 -o type=docker,dest=dist/rss-reader-linux-arm64.tar .
	@echo "跨平台 Docker 构建完成，产物在 dist/ 目录"

# 跨平台构建并推送到镜像仓库，用法: make docker-build-cross-push IMAGE=your-registry/rss-reader:latest
docker-build-cross-push:
	@if [ -z "$(IMAGE)" ]; then echo "请指定 IMAGE，例如: make docker-build-cross-push IMAGE=your-registry/rss-reader:latest"; exit 1; fi
	docker buildx build --platform linux/amd64,linux/arm64 -t $(IMAGE) --push .

# 单平台 Docker 构建
docker-build-linux-amd64:
	docker buildx build --platform linux/amd64 -t rss-reader:linux-amd64 --load .

docker-build-linux-arm64:
	docker buildx build --platform linux/arm64 -t rss-reader:linux-arm64 --load .
