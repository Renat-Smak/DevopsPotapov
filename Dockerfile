FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/pharmacy ./cmd/pharmacy

FROM alpine:3.23
COPY --from=build /out/pharmacy /usr/local/bin/pharmacy
USER 65532:65532
EXPOSE 8080 9090
ENTRYPOINT ["/usr/local/bin/pharmacy"]
