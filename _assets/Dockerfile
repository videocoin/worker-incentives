FROM golang:1.14-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git
WORKDIR /build/
ADD . .

RUN make build

FROM alpine:latest

COPY --from=builder /build/build/server /

WORKDIR /
ENTRYPOINT ["/server"]