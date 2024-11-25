FROM golang:1.23-alpine AS builder

WORKDIR /app

ADD . /app

RUN go build cmd/sidewinder/main.go

FROM alpine:3.20

ENV SIDEWINDER_CONFIG_FILE="/app/sidewinder.toml"
ENV SIDEWINDER_DATA_DIR="/data"
ENV SIDEWINDER_TICK_RATE="30m"

WORKDIR /app

COPY --from=builder /app/main /app/sidewinder

CMD ["/app/sidewinder", "run"]
