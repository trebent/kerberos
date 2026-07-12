FROM golang:1.26.5@sha256:079e59808d2d252516e27e3f3a9c003740dee7f75e55aa71528766d52bcfc16a AS builder

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

FROM gcr.io/distroless/static-debian12:nonroot@sha256:b7bb25d9f7c31d2bdd1982feb4dafcaf137703c7075dbe2febb41c24212b946f AS runtime

USER nonroot:nonroot

COPY --chown=nonroot:nonroot --from=build /kerberos /kerberos
COPY --chown=nonroot:nonroot --from=build openapi/ /krb-oas/
COPY --chown=nonroot:nonroot --from=build poc/poc.json /poc.json

ARG VERSION="unset"

ENV VERSION=${VERSION}
ENV OAS_DIRECTORY=/krb-oas

EXPOSE 30000

ENTRYPOINT [ "/kerberos" ]
