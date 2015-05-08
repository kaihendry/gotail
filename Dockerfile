FROM golang

ADD . /go/src/github.com/nino-k/gotail

RUN go install github.com/nino-k/gotail

ENTRYPOINT /go/bin/gotail

EXPOSE 8080
