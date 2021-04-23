FROM golang:1.16 AS builder
COPY . /opt/build
WORKDIR /opt/build
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o incentives ./cmd


FROM registry.videocoin.net/worker-availability/job:4fed7bd3e90eab6c44242b661d05695147f84266
RUN mkdir /opt/incentives
RUN apk update && apk add bash
WORKDIR /opt/incentives
COPY --from=builder /opt/build/incentives /opt/incentives/incentives
CMD ["./opt/incentives/incentivescron.sh"]