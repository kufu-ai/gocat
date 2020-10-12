FROM golang:1.15.0-alpine AS build

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

