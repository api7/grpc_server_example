FROM golang:1.14.10 AS build-env

RUN rm -rf /etc/localtime \
    && ln -s /usr/share/zoneinfo/Hongkong /etc/localtime \
    && dpkg-reconfigure -f noninteractive tzdata

WORKDIR /build
COPY . .
RUN GOPROXY=https://goproxy.cn,direct go build -o grpc_server_example main.go

FROM centos:centos7

WORKDIR /grpc_server_example
COPY --from=build-env /build/grpc_server_example bin/
COPY --from=build-env /build/t t/

WORKDIR /grpc_server_example/bin
EXPOSE 50051
EXPOSE 50052
ENTRYPOINT ["/grpc_server_example/bin/grpc_server_example"]
