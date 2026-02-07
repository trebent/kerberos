FROM golang:1.25.7@sha256:11dcd46e924310d4a38f4994c5783eeb2baf99e3999312bec1160185cab8980f AS builder

WORKDIR /

FROM builder AS deps

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
  go mod download

FROM deps AS build

COPY . .

ENV GOOS=linux
ENV CGO_ENABLED=0

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  go build -trimpath -ldflags="-s -w" -o echo ./cmd/echo

FROM gcr.io/distroless/static-debian12:nonroot@sha256:cba10d7abd3e203428e86f5b2d7fd5eb7d8987c387864ae4996cf97191b33764 AS runtime

COPY --from=build /echo /echo

ARG VERSION="unset"

ENV VERSION=${VERSION}

EXPOSE 15000

ENTRYPOINT [ "/echo" ]
