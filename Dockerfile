FROM golang:bookworm

WORKDIR /src
ADD . ./

RUN go build -o achilles

ENTRYPOINT ["./achilles"]
