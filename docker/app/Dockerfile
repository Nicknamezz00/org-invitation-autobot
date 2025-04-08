FROM golang:alpine

RUN go env -w GOPROXY=https://mirrors.aliyun.com/goproxy,direct
RUN go install github.com/cespare/reflex@v0.3.1
COPY reflex.conf /

COPY start.sh /

ENTRYPOINT ["reflex", "-c", "/reflex.conf"]
