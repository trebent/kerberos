FROM golang:1.27rc2@sha256:d901fb8d9f6754d898c85bb9331ad76dbe039103b01cf0c068bb31f92e019788 AS builder

WORKDIR /

FROM builder AS deps

COPY go.mod go.sum ./

RUN go mod download

FROM deps AS build

COPY . .

ENV GOOS=linux
ENV CGO_ENABLED=0

RUN --mount=type=cache,target=/root/.cache/go-build \
  go build -trimpath -ldflags="-s -w" -o echo ./cmd/echo

FROM gcr.io/distroless/static-debian12:nonroot@sha256:1b7b9f0f0e0a1d2155f531db587cc48ec26aaf97ab64364225f5bf18a054e66a AS runtime

USER nonroot:nonroot

COPY --chown=nonroot:nonroot --from=build /echo /echo

ARG VERSION="unset"

ENV VERSION=${VERSION}

EXPOSE 15000

ENTRYPOINT [ "/echo" ]
