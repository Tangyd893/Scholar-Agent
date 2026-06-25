# Agent Core Dockerfile — 多阶段构建
# 从仓库根目录构建：docker build -f agent-core/Dockerfile -t scholar-agent-core .

FROM golang:alpine AS builder
WORKDIR /app

COPY go.work ./
COPY go.mod go.sum ./
COPY gateway/go.mod gateway/go.sum ./gateway/
COPY agent-core/go.mod agent-core/go.sum ./agent-core/
COPY tool-service/go.mod tool-service/go.sum ./tool-service/
COPY proto/gen/go.mod proto/gen/go.sum ./proto/gen/
COPY scripts/go.mod scripts/go.sum ./scripts/

RUN go work sync
COPY . .

RUN CGO_ENABLED=0 go build -o /agent-core ./agent-core/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /agent-core /agent-core
EXPOSE 50051
ENTRYPOINT ["/agent-core"]
