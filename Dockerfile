FROM golang:1.19.4-bullseye AS build

RUN apt update && apt install git libc-dev

WORKDIR /src

COPY go.* ./
RUN go mod download

COPY . .

RUN go build -o ./gocat

FROM alpine

WORKDIR /src
COPY --from=build /src/gocat /src/gocat

CMD /src/gocat

