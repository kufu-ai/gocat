FROM golang:1.19.4-bullseye AS build

RUN apt update && apt install -y git libc-dev

WORKDIR /src

COPY go.* ./
RUN go mod download

COPY . .

RUN go build -o ./gocat

FROM debian:bullseye AS deps

RUN apt update && apt install -y curl

ENV KANVAS_VERSION=0.8.0

RUN curl -LO https://github.com/davinci-std/kanvas/releases/download/v${KANVAS_VERSION}/kanvas_${KANVAS_VERSION}_linux_amd64.tar.gz \
    && tar -xzf kanvas_${KANVAS_VERSION}_linux_amd64.tar.gz \
    && mv kanvas /usr/local/bin/kanvas \
    && rm kanvas_${KANVAS_VERSION}_linux_amd64.tar.gz

FROM debian:bullseye

RUN apt update && apt install -y git

WORKDIR /src
COPY --from=build /src/gocat /src/gocat
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=deps /usr/local/bin/kanvas /usr/local/bin/kanvas

CMD /src/gocat
