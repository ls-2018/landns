FROM golang:latest AS builder

RUN go env -w GOPATH=/tmp/.gopath
RUN go env -w GO111MODULE=on 
RUN go env -w CGO_ENABLED=1 
RUN go env -w GOPROXY=https://goproxy.cn/,direct
WORKDIR /dns
COPY . .

RUN go build -o /landns .


FROM centos:8

COPY --from=builder /landns /landns
COPY resolv.conf /etc/resolv.conf
EXPOSE 53/udp
EXPOSE 9353/tcp
ENTRYPOINT ["/bin/sh","-c"]
