FROM golang:1.16 AS builder
COPY . /opt/build
WORKDIR /opt/build
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o incentives ./cmd
RUN cd cron && CGO_ENABLED=0 GOOS=linux go build -o cron

FROM registry.videocoin.net/worker-availability/job:4fed7bd3e90eab6c44242b661d05695147f84266 as availability

FROM golang:1.13.15-alpine3.12
RUN mkdir /opt/incentives
RUN apk update && apk add bash
WORKDIR /opt/incentives
COPY --from=availability /job /opt/incentives/job
COPY --from=builder /opt/build/incentives /opt/incentives/incentives
COPY --from=builder /opt/build/cron/cron /opt/incentives/cron
CMD ["./opt/incentives/incentivescron.sh"]