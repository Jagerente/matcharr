FROM --platform=$BUILDPLATFORM node:26-alpine AS web
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /app/web/dist ./web/dist
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
  go build -trimpath -ldflags="-s -w" -o /matcharr ./cmd/matcharr

# alpine over scratch: ca-certificates for TMDB TLS, wget for the healthcheck
FROM alpine:3.24
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /matcharr /usr/local/bin/matcharr

ENV MATCHARR__CONFIG__DIR=/config
VOLUME /config
EXPOSE 8585

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
  CMD wget -q --spider http://localhost:8585/healthz || exit 1

ENTRYPOINT ["matcharr"]
