FROM golang:1.27@sha256:0ecdc2a9f6156af6451080bfe3d8382a662fcc4e209608c6f919e643453514c1 AS builder

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

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab AS runtime

USER nonroot:nonroot

COPY --chown=nonroot:nonroot --from=build /echo /echo

ARG VERSION="unset"

ENV VERSION=${VERSION}

EXPOSE 15000

ENTRYPOINT [ "/echo" ]
