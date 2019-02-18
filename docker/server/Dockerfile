FROM golang:1.11-alpine3.7


WORKDIR ${GOPATH}/src/github.com/shiotomo/codecaramel
ADD . ${GOPATH}/src/github.com/shiotomo/codecaramel

RUN apk update \
  && apk add --no-cache git bash\
  && go get -u github.com/golang/dep/cmd/dep \
  && dep ensure

CMD go run server.go
