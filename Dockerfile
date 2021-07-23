# syntax=docker/dockerfile:1.2
ARG GO_VERSION=1.16

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine as builder
ARG GIT_REF
WORKDIR /go/src/github.com/lazyshot/emeter-exporter
COPY . .
ARG TARGETPLATFORM
RUN GOARCH=${TARGETPLATFORM} CGO_ENABLED=0 go build -a -installsuffix cgo .

FROM --platform=$BUILDPLATFORM alpine:latest
RUN apk --no-cache add ca-certificates rtl-sdr
WORKDIR /app
COPY --from=builder /go/src/github.com/lazyshot/emeter-exporter/emeter-exporter .
COPY start.sh .
CMD ["./start.sh"]
