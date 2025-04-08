FROM golang:alpine AS builder

RUN mkdir -p /internal
COPY . /app
WORKDIR /app

RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o main -ldflags="-w -s" .


FROM alpine AS runner
WORKDIR /app
COPY --from=builder /app/main .
CMD ["./main"]