FROM golang:1.24 AS builder

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct

RUN mkdir -p /app
WORKDIR /app

COPY . .
RUN go mod tidy

RUN GOOS=linux GOARCH=amd64 go build -o main .

FROM alpine as RUNNER

WORKDIR /app
COPY --from=builder /app .

CMD ["/app/main"]
