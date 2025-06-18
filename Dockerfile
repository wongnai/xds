FROM golang:1.24 AS builder
ENV GOTOOLCHAIN=local
RUN go telemetry off
COPY . /build
WORKDIR /build
RUN make

FROM debian:bookworm-slim
USER 0
COPY --from=builder /build/.bin/xds /opt/xds
USER 1000
WORKDIR /
CMD ["/opt/xds"]
