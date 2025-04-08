FROM golang:alpine AS builder

RUN mkdir -p /app
COPY . .
WORKDIR /

RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o main .


FROM alpine AS runner
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/config ./config
CMD ["./main"]


# 打包依赖阶段使用golang作为基础镜像
FROM golang:alpine as builder

# 启用go module
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct

WORKDIR /app

COPY . .

RUN GOOS=linux GOARCH=amd64 go build -o main .

# 运行阶段指定scratch作为基础镜像
FROM alpine as RUNNER

WORKDIR /app

COPY --from=builder /app .

ENV GIN_MODE=release

ENTRYPOINT ["./main"]