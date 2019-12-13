# Building stage
FROM golang:1.12-alpine3.9

RUN apk add --update make

WORKDIR /go/src/github.com/duyanghao/coredns-sync

# Source code, building tools and dependences
COPY cmd /go/src/github.com/duyanghao/coredns-sync/cmd
COPY Makefile /go/src/github.com/duyanghao/coredns-sync

COPY go.mod /go/src/github.com/duyanghao/coredns-sync
COPY go.sum /go/src/github.com/duyanghao/coredns-sync
COPY vendor /go/src/github.com/duyanghao/coredns-sync/vendor

ENV CGO_ENABLED 0
ENV GOOS linux
ENV GO111MODULE=on

ENV TIMEZONE "Asia/Shanghai"

RUN make src.build
RUN mv build/coredns-sync/coredns-sync /go/bin

# Production stage
FROM alpine:3.9
WORKDIR /go/bin

# copy the go binaries from the building stage
COPY --from=0 /go/bin /go/bin

# copy the config files from the current working dir
COPY examples /coredns-sync/examples

ENTRYPOINT ["./coredns-sync", "-c", "/coredns-sync/examples/config.yml", "-v", "5", "-logtostderr", "true"]
