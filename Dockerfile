# 多阶段构建：产出静态链接二进制
FROM golang:1.22-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/sniproxy .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/sniproxy /usr/local/bin/sniproxy
COPY config.yaml /etc/sniproxy/config.yaml

# 容器内默认 root，便于绑定 80/443；生产可用 --user 或改高位端口
EXPOSE 80 443 9100

ENTRYPOINT ["/usr/local/bin/sniproxy"]
CMD ["-c", "/etc/sniproxy/config.yaml"]
