### build go executable
FROM golang:1.19

COPY . /go/src/alert-namespace-silencer
WORKDIR /go/src/alert-namespace-silencer
RUN go build -o alert-namespace-silencer

### final image
FROM alpine:latest

WORKDIR /

RUN apk add --no-cache tini bash
COPY --from=0 /go/src/alert-namespace-silencer/alert-namespace-silencer /alert-namespace-silencer
RUN chmod 755 /alert-namespace-silencer

ENTRYPOINT ["/sbin/tini", "--", "/alert-namespace-silencer"]
