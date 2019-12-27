# Building stage
FROM golang:1.12-alpine3.9

RUN apk add --update make

WORKDIR /go/src/github.com/duyanghao/cluster-coredns-controller

# Source code, building tools and dependences
COPY cmd /go/src/github.com/duyanghao/cluster-coredns-controller/cmd
COPY pkg /go/src/github.com/duyanghao/cluster-coredns-controller/pkg
COPY Makefile /go/src/github.com/duyanghao/cluster-coredns-controller

COPY go.mod /go/src/github.com/duyanghao/cluster-coredns-controller
COPY go.sum /go/src/github.com/duyanghao/cluster-coredns-controller
COPY vendor /go/src/github.com/duyanghao/cluster-coredns-controller/vendor

ENV CGO_ENABLED 0
ENV GOOS linux
ENV GO111MODULE=on

ENV TIMEZONE "Asia/Shanghai"

RUN make src.build
RUN mv build/cluster-coredns-controller/cluster-coredns-controller /go/bin

# Production stage
FROM alpine:3.9
WORKDIR /go/bin

# copy the go binaries from the building stage
COPY --from=0 /go/bin /go/bin

# copy the config files from the current working dir
COPY examples /cluster-coredns-controller/examples

ENTRYPOINT ["./cluster-coredns-controller", "-c", "/cluster-coredns-controller/examples/config.yml", "-v", "4", "-logtostderr", "true"]
