FROM golang:1.12.5-alpine3.9 as builder

RUN apk update && apk upgrade && apk add git gcc libc-dev && \
    go get -u cloud.google.com/go/bigtable && \
    go get -u github.com/stretchr/testify && \
    go get -u github.com/google/btree

ADD *.go /go/bin/

ENV BIGTABLE_EMULATOR_HOST=localhost:9035

RUN go build /go/bin/bigtable-emulator.go && \
    /go/bigtable-emulator & \
    sleep 1 && \
    go test -v /go/bin/bigtable-emulator_test.go


FROM alpine:3.9

COPY --from=builder /go/bigtable-emulator /

ENTRYPOINT ["/bigtable-emulator"]

EXPOSE 9035
