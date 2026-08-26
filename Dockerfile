# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS build

WORKDIR /src

# 先复制 go.mod，利用 Docker 层缓存；本项目无第三方依赖，仍保持标准分层。
COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/cssd-instrument-trace-service .

FROM alpine:3.20

RUN addgroup -S app && adduser -S -G app app
RUN mkdir -p /app/data && chown -R app:app /app

WORKDIR /app
COPY --from=build /out/cssd-instrument-trace-service /usr/local/bin/cssd-instrument-trace-service

USER app
ENV PORT=8080
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null "http://127.0.0.1:${PORT}/healthz" || exit 1

ENTRYPOINT ["/usr/local/bin/cssd-instrument-trace-service"]
