FROM golang:1.26-alpine3.23 AS builder
WORKDIR /src
COPY go.mod main.go main_test.go ./
RUN go test ./... && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/counter .

FROM alpine:3.23 AS packer
RUN apk add --no-cache upx
COPY --from=builder /out/counter /out/counter
RUN upx --best /out/counter && upx -t /out/counter

FROM alpine:3.19 AS runtime
USER 65532:65532
EXPOSE 8080
HEALTHCHECK --interval=2s --timeout=1s --start-period=2s --retries=3 \
  CMD wget -q --spider http://127.0.0.1:8080/health || exit 1
ENTRYPOINT ["/usr/local/bin/counter"]

FROM runtime AS plain
COPY --from=builder /out/counter /usr/local/bin/counter

FROM runtime AS packed
COPY --from=packer /out/counter /usr/local/bin/counter

FROM packed AS unhealthy
ENV LAB9_HEALTH_FAIL=1

FROM packed AS release
