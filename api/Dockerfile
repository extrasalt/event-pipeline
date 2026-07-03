FROM golang:1.25 AS builder
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
ARG TARGETARCH
RUN case "$TARGETARCH" in \
      amd64) URL="https://github.com/chdb-io/chdb-core/releases/download/v26.5.0/linux-x86_64-libchdb.tar.gz" ;; \
      arm64) URL="https://github.com/chdb-io/chdb-core/releases/download/v26.5.0/linux-aarch64-libchdb.tar.gz" ;; \
      *) echo "unsupported arch: $TARGETARCH"; exit 1 ;; \
    esac && \
    wget -qO /tmp/libchdb.tar.gz "$URL" && \
    tar -xzf /tmp/libchdb.tar.gz -C /tmp && \
    mv /tmp/libchdb.so /usr/lib/libchdb.so && \
    rm -f /tmp/libchdb.tar.gz /tmp/chdb.h
COPY api/ /build/api/
COPY pipeline/ /build/pipeline/
WORKDIR /build/api
RUN go build -o /api-server ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates libstdc++6 && rm -rf /var/lib/apt/lists/*
COPY --from=builder /api-server /api-server
COPY --from=builder /usr/lib/libchdb.so /usr/lib/libchdb.so
ENV LD_LIBRARY_PATH=/usr/lib
EXPOSE 8081
CMD ["/api-server"]
