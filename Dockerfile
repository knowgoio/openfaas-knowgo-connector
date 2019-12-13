FROM golang:1.13.1-alpine3.10 as build

ENV CGO_ENABLED=0
ENV GO111MODULE=on

RUN mkdir -p /go/src/github.com/knowgoio/openfaas-knowgo-connector
WORKDIR /go/src/github.com/knowgoio/openfaas-knowgo-connector

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY main.go    .

# Run a gofmt and exclude all vendored code.
RUN test -z "$(gofmt -l $(find . -type f -name '*.go' -not -path "./vendor/*"))"

RUN go test -v ./...

# Stripping via -ldflags "-s -w" 
RUN CGO_ENABLED=0 GOOS=linux \
    go build --ldflags "-s -w" -a -installsuffix cgo -o /usr/bin/producer . && \
    go test $(go list ./... | grep -v /vendor/) -cover

FROM alpine:3.10 as ship
RUN apk add --no-cache ca-certificates

COPY --from=build /usr/bin/producer /usr/bin/producer
WORKDIR /root/

CMD ["/usr/bin/producer"]
