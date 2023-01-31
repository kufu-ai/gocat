FROM golang:1.19.4-bullseye AS build

RUN apt update && apt install -y git libc-dev

WORKDIR /src

COPY go.* ./
RUN go mod download

COPY . .

RUN go build -o ./gocat

FROM debian:bullseye

WORKDIR /src
COPY --from=build /src/gocat /src/gocat

CMD /src/gocat

