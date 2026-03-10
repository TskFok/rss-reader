# 支持多平台构建: docker buildx build --platform linux/amd64,linux/arm64 -t rss-reader:latest .
# Stage 1: 构建前端
FROM node:20-alpine AS frontend
WORKDIR /web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: 构建 Go 二进制
FROM golang:1.21-alpine AS backend
ARG TARGETOS=linux
ARG TARGETARCH=amd64
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /web/dist ./cmd/server/static
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o rss-reader ./cmd/server

# Stage 3: 运行
FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=backend /app/rss-reader /rss-reader
COPY config.example.yaml /config.yaml
WORKDIR /
EXPOSE 8080
ENTRYPOINT ["/rss-reader"]
