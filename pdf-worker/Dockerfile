FROM golang:alpine AS builder
WORKDIR /app

COPY go.work ./
COPY go.mod go.sum ./
COPY gateway/go.mod gateway/go.sum ./gateway/
COPY agent-core/go.mod agent-core/go.sum ./agent-core/
COPY tool-service/go.mod tool-service/go.sum ./tool-service/
COPY pdf-worker/go.mod pdf-worker/go.sum ./pdf-worker/
COPY proto/gen/go.mod proto/gen/go.sum ./proto/gen/
COPY scripts/go.mod scripts/go.sum ./scripts/

RUN go work sync
COPY . .

RUN CGO_ENABLED=0 go build -o /pdf-worker ./pdf-worker/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /pdf-worker /pdf-worker
ENTRYPOINT ["/pdf-worker"]
