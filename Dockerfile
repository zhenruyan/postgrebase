# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM node:22-alpine AS ui-builder

WORKDIR /src/ui

COPY ui/package.json ui/package-lock.json ./
RUN npm ci

COPY ui/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.26.2-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
COPY vendor/ ./vendor/
COPY . .
COPY --from=ui-builder /src/ui/dist ./ui/dist

ARG TARGETOS=linux
ARG TARGETARCH

ENV CGO_ENABLED=0

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -mod=vendor \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/pb \
    ./build/

FROM alpine:3.20

RUN apk add --no-cache ca-certificates \
    && addgroup -S pocketbase \
    && adduser -S -G pocketbase pocketbase \
    && mkdir -p /pb/pb_data /pb/pb_public \
    && chown -R pocketbase:pocketbase /pb

WORKDIR /pb

COPY --from=builder /out/pb /usr/local/bin/pb

USER pocketbase

EXPOSE 8090
VOLUME ["/pb/pb_data"]

ENTRYPOINT ["pb"]
CMD ["serve", "--http=0.0.0.0:8090", "--dir=/pb/pb_data", "--publicDir=/pb/pb_public"]
