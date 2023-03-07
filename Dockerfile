FROM golang:alpine as builder

RUN mkdir /build
ADD . /build/
WORKDIR /build

# Build the binary
RUN CGO_ENABLED=0 go build -o envoy-xds-server ./cmd/server/main.go

# Copy into scratch
FROM scratch
COPY --from=builder /build/xds-control-server /bin/xds-control-server
CMD ["/bin/xds-control-server"]
