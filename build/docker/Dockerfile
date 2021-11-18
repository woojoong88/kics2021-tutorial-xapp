# docker build . -f build/docker/Dockerfile -t woojoong/sample-xapp:latest

FROM onosproject/golang-build:v0.6.7 as build

ENV GO111MODULE=on
ARG ONOS_MAKE_TARGET=build

COPY go.mod go.sum /go/src/sample-xapp/
COPY cmd/ /go/src/sample-xapp/cmd/
COPY pkg/ /go/src/sample-xapp/pkg/
COPY vendor/ /go/src/sample-xapp/vendor/

RUN cd /go/src/sample-xapp && GOFLAGS=-mod=vendor go build -o build/_output/sample-xapp ./cmd

FROM alpine:3.11
RUN apk add libc6-compat

USER nobody

COPY --from=build /go/src/sample-xapp/build/_output/sample-xapp /usr/local/bin/sample-xapp

ENTRYPOINT ["sample-xapp"]