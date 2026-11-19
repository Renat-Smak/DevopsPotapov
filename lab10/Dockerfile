FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/pharmacy ./cmd/pharmacy

FROM alpine:3.23
COPY --from=build /out/pharmacy /usr/local/bin/pharmacy
USER 65532:65532
EXPOSE 8080 9090
HEALTHCHECK --interval=2s --timeout=2s --start-period=2s --retries=8 \
  CMD wget -q --spider http://127.0.0.1:8080/health || exit 1
ENTRYPOINT ["/usr/local/bin/pharmacy"]
