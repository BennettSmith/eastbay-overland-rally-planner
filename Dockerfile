## Multi-stage build for the Go API service.
##
## Build:
##   docker build -t ebo-api .
##
## Run (example):
##   docker run --rm -p 8080:8080 -e PORT=8080 ebo-api

FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

WORKDIR /src

# Certs are useful for modules download and any TLS calls during tests/build.
RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH

# Build a small static binary.
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Distroless runtime (includes CA certs) running as nonroot.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/api /api

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/api"]


