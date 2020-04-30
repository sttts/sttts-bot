FROM golang:1.13 AS builder
WORKDIR /go/src/github.com/sttts/sttts-bot
COPY . .
RUN make

FROM fedora:32
COPY --from=builder /go/src/github.com/openshift/sttts/sttts-bot /usr/bin/
ENTRYPOINT ["/usr/bin/sttts-bot"]
