FROM --platform=$BUILDPLATFORM node:25.9.0-bookworm-slim AS frontend-builder

WORKDIR /src/frontend

RUN corepack enable && corepack prepare pnpm@10.30.3 --activate

COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY frontend/ ./
RUN pnpm build

FROM --platform=$BUILDPLATFORM golang:1.26.3 AS backend-builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /src/frontend/dist ./frontend/dist

ARG NAME=momoko
ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build \
    -trimpath \
    -ldflags="-s -w -X main.Name=${NAME} -X main.Version=${VERSION}" \
    -o /out/momoko \
    ./cmd/momoko

FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    netbase \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get autoremove -y \
    && apt-get autoclean -y

WORKDIR /app

COPY --from=backend-builder /out/momoko /app/momoko
COPY configs /app/configs

EXPOSE 22633 22733

VOLUME ["/app/data", "/app/configs"]

CMD ["/app/momoko", "-conf", "/app/configs"]
