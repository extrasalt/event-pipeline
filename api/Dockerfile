FROM golang:1.25-alpine AS builder
COPY api/ /build/api/
COPY pipeline/ /build/pipeline/
WORKDIR /build/api
RUN CGO_ENABLED=0 go build -o /api-server ./cmd/server

FROM alpine:3.20
COPY --from=builder /api-server /api-server
EXPOSE 8081
CMD ["/api-server"]
