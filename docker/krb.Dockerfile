FROM golang:1.26.5@sha256:ae5a2316d12f3e78fd99177dad452e6ad4f240af2d71d57b480c3477f250fec6 AS builder

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

FROM gcr.io/distroless/static-debian12:nonroot@sha256:aef9602f8710ec12bde19d593fed1f76c708531bb7aba205110f1029786ead7b AS runtime

USER nonroot:nonroot

COPY --chown=nonroot:nonroot --from=build /kerberos /kerberos
COPY --chown=nonroot:nonroot --from=build openapi/ /krb-oas/
COPY --chown=nonroot:nonroot --from=build poc/poc.json /poc.json

ARG VERSION="unset"

ENV VERSION=${VERSION}
ENV OAS_DIRECTORY=/krb-oas

EXPOSE 30000

ENTRYPOINT [ "/kerberos" ]
