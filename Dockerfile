# 多阶段构建：第一阶段编译,第二阶段瘦身
FROM golang:1.23-alpine AS builder

WORKDIR /app

# 利用缓存
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO=0 产出纯静态二进制,alpine 镜像也能跑
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/server .

# ── 运行时镜像: 用 alpine 就行,只需 ca-certificates (调 HTTPS AI) ──
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app

COPY --from=builder /out/server /app/server
COPY config/config.yaml /app/config/config.yaml

EXPOSE 8080

CMD ["/app/server"]