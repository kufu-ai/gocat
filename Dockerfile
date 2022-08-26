FROM golang:1.19.0-alpine3.16 AS build

RUN apk add --update git libc-dev

WORKDIR /src

COPY go.* ./
RUN go mod download

COPY . .

RUN go build -o ./gocat

FROM alpine

WORKDIR /src
COPY --from=build /src/gocat /src/gocat

CMD /src/gocat

