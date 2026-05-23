# syntax=docker/dockerfile:1.7

FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/capy ./cmd/capy

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/capy /usr/local/bin/capy
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/capy"]
CMD ["help"]
