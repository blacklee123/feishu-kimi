FROM golang:1.22-alpine as go-builder


WORKDIR /app
COPY pkg pkg
COPY main.go main.go
COPY go.mod go.mod
COPY go.sum go.sum

RUN GOPROXY=https://goproxy.cn go mod download

RUN CGO_ENABLED=0 go build -ldflags '-w -s' -a -o  feishu-kimi

FROM alpine:3.20
RUN apk --no-cache add tzdata && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo "Asia/Shanghai" >/etc/timezone

WORKDIR /app

# RUN apk add --no-cache bash
COPY --from=go-builder /app/feishu-kimi /app
EXPOSE 9000
ENTRYPOINT ["/app/feishu-kimi"]
