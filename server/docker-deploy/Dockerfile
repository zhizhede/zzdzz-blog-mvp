# 后端运行时镜像 (基础 + 编译好的二进制)
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app

COPY server-linux-amd64 /app/server
COPY config/config.yaml /app/config/config.yaml
# config.local.yaml 不进镜像,部署时挂载

EXPOSE 8080

CMD ["/app/server"]