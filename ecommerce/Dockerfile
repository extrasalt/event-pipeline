FROM node:22-alpine AS frontend
ENV CI=true
RUN corepack enable
WORKDIR /build
COPY ecommerce/app/package.json ecommerce/app/pnpm-lock.yaml ecommerce/app/pnpm-workspace.yaml ./
RUN pnpm install
COPY ecommerce/app/ ./
COPY script/tracker.js ./public/script/tracker.js
RUN pnpm run build

FROM golang:1.25-alpine AS builder
COPY ecommerce/ /build/ecommerce/
WORKDIR /build/ecommerce
RUN CGO_ENABLED=0 go build -o /bffe .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /bffe /bffe
COPY --from=frontend /build/dist /app/dist
COPY script/tracker.js /app/dist/script/tracker.js
EXPOSE 8080
CMD ["/bffe"]
