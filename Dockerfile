FROM golang:1.26.3@sha256:cc9a5d7a008cfe2cbc7ffc752b0d6636ad30fc16e4a648d2e4aac00fd8b25ca3 AS builder

WORKDIR /

FROM builder AS deps

COPY go.mod go.sum ./

RUN go mod download

FROM deps AS build

COPY . .

ENV GOOS=linux
ENV CGO_ENABLED=0

RUN --mount=type=cache,target=/root/.cache/go-build \
  go build -trimpath -ldflags="-s -w" -o kerberos .

FROM gcr.io/distroless/static-debian12:nonroot@sha256:a9329520abc449e3b14d5bc3a6ffae065bdde0f02667fa10880c49b35c109fd1 AS runtime

COPY --from=build /kerberos /kerberos
COPY --from=build openapi/ /oas/

ARG VERSION="unset"

ENV VERSION=${VERSION}

EXPOSE 30000

ENTRYPOINT [ "/kerberos" ]
