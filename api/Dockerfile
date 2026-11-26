# syntax=docker/dockerfile:1

# Keep in sync with the `go` directive in go.mod.
ARG GO_VERSION=1.27.4

FROM golang:${GO_VERSION} AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Distroless has no shell or curl: health checks run `/api healthcheck`.
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/api /api
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/api"]
