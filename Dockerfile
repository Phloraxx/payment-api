# syntax=docker/dockerfile:1.7

FROM node:22.23.1-bookworm-slim AS web-build
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci
COPY web ./web
RUN npm run typecheck:v3 && npm run build:v3

FROM golang:1.25.13-bookworm AS go-build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
COPY --from=web-build /src/internal/web/dist ./internal/web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w" -o /out/paygate ./cmd/payment-api
RUN mkdir -p /out/pb_data

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=go-build --chown=nonroot:nonroot /out/paygate /app/paygate
COPY --from=go-build --chown=nonroot:nonroot /out/pb_data /app/pb_data
COPY --chown=nonroot:nonroot LICENSE NOTICE /app/
USER nonroot:nonroot
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["/app/paygate", "healthcheck"]
ENTRYPOINT ["/app/paygate"]
CMD ["serve", "--http=0.0.0.0:3000"]
