FROM golang:1.18-alpine AS builder
ENV GOPRIVATE=github.com/videocoin/*
ARG BOT_USER="nothing"
ARG BOT_PASSWORD="nothing"
RUN apk add --no-cache ca-certificates git
RUN git config --global url."https://${BOT_USER}:${BOT_PASSWORD}@github.com".insteadOf "https://github.com"
COPY . /opt/build
WORKDIR /opt/build
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod -o incentives ./cmd
RUN cd cron && CGO_ENABLED=0 GOOS=linux go build -o cron

FROM registry.videocoin.net/worker-availability/job:9be774d26aea0eab58c047f776e82870436ba259 as availability

FROM golang:1.13.15-alpine3.12
RUN mkdir /opt/incentives
RUN apk update && apk add bash
WORKDIR /opt/incentives
COPY --from=availability /job /opt/incentives/job
COPY --from=builder /opt/build/incentives /opt/incentives/incentives
COPY --from=builder /opt/build/cron/cron /opt/incentives/cron
CMD ["./opt/incentives/incentivescron.sh"]