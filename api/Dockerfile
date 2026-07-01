FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY api/ api/
COPY pipeline/ pipeline/
RUN CGO_ENABLED=0 go build -o /api-server ./api/cmd/server

FROM alpine:3.20
COPY --from=builder /api-server /api-server
EXPOSE 8081
CMD ["/api-server"]
