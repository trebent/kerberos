FROM golang:1.26.1@sha256:e2ddb153f786ee6210bf8c40f7f35490b3ff7d38be70d1a0d358ba64225f6428 AS builder

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

FROM gcr.io/distroless/static-debian12:nonroot@sha256:a9329520abc449e3b14d5bc3a6ffae065bdde0f02667fa10880c49b35c109fd1 AS runtime

COPY --from=build /kerberos /kerberos
COPY --from=build openapi/ /oas/

ARG VERSION="unset"

ENV VERSION=${VERSION}

EXPOSE 30000

ENTRYPOINT [ "/kerberos" ]
