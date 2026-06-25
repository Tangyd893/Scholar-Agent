# Gateway Dockerfile — 多阶段构建
# 从仓库根目录构建：docker build -f gateway/Dockerfile -t scholar-gateway .

# ---- Build Stage ----
FROM golang:alpine AS builder
WORKDIR /app

# 复制 go.work 和模块定义（利用 Docker 缓存层）
COPY go.work ./
COPY go.mod go.sum ./
COPY gateway/go.mod gateway/go.sum ./gateway/
COPY agent-core/go.mod agent-core/go.sum ./agent-core/
COPY tool-service/go.mod tool-service/go.sum ./tool-service/
COPY proto/gen/go.mod proto/gen/go.sum ./proto/gen/
COPY scripts/go.mod scripts/go.sum ./scripts/

# 下载依赖
RUN go work sync

# 复制源代码
COPY . .

# 构建
RUN CGO_ENABLED=0 go build -o /gateway ./gateway/

# ---- Run Stage ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /gateway /gateway
EXPOSE 8080
ENTRYPOINT ["/gateway"]
