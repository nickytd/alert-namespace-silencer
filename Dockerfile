### build go executable
FROM golang:1.15

COPY . /go/src/alert-namespace-silencer
WORKDIR /go/src/alert-namespace-silencer
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -o alert-namespace-silencer

### final image
FROM alpine:latest

WORKDIR /

RUN apk add --no-cache tini bash
COPY --from=0 /go/src/alert-namespace-silencer/alert-namespace-silencer /alert-namespace-silencer
RUN chmod 755 /alert-namespace-silencer

ENTRYPOINT ["/sbin/tini", "--", "/alert-namespace-silencer"]
