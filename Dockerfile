FROM golang:1.26.0@sha256:21d68a382b318dde54771baf5e04393e0930d077583c8010e4fd83adae328eda AS builder

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
  go build -trimpath -ldflags="-s -w" -o kerberos .

FROM gcr.io/distroless/static-debian12:nonroot@sha256:cba10d7abd3e203428e86f5b2d7fd5eb7d8987c387864ae4996cf97191b33764 AS runtime

COPY --from=build /kerberos /kerberos
COPY --from=build openapi/ /oas/

ARG VERSION="unset"

ENV VERSION=${VERSION}

EXPOSE 30000

ENTRYPOINT [ "/kerberos" ]
