FROM golang:1.19.4-bullseye AS build

RUN apt update && apt install -y git libc-dev

WORKDIR /src

COPY go.* ./
RUN go mod download

COPY . .

RUN go build -o ./gocat

FROM debian:bullseye AS deps

RUN apt update && apt install -y curl

ENV KANVAS_VERSION=0.11.1

RUN curl -LO https://github.com/davinci-std/kanvas/releases/download/v${KANVAS_VERSION}/kanvas_${KANVAS_VERSION}_linux_amd64.tar.gz \
    && tar -xzf kanvas_${KANVAS_VERSION}_linux_amd64.tar.gz \
    && mv kanvas /usr/local/bin/kanvas \
    && rm kanvas_${KANVAS_VERSION}_linux_amd64.tar.gz

ENV KUSTOMIZE_VERSION=5.3.0

RUN curl -LO https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/v${KUSTOMIZE_VERSION}/kustomize_v${KUSTOMIZE_VERSION}_linux_amd64.tar.gz \
  && tar xzf ./kustomize_v${KUSTOMIZE_VERSION}_linux_amd64.tar.gz \
  && mv kustomize /usr/local/bin/kustomize \
  && rm kustomize_v${KUSTOMIZE_VERSION}_linux_amd64.tar.gz \
  && chmod +x /usr/local/bin/kustomize

FROM debian:bullseye

RUN apt update && apt install -y git

WORKDIR /src
COPY --from=build /src/gocat /src/gocat
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=deps /usr/local/bin/kustomize /usr/local/bin/kustomize
COPY --from=deps /usr/local/bin/kanvas /usr/local/bin/kanvas

CMD /src/gocat
