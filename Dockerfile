FROM golang:1.16.7-alpine AS build

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

