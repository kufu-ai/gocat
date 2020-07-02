FROM golang:1.14.3-stretch

RUN mkdir /bot
WORKDIR /bot

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .
RUN go build

CMD ./slack-bot
