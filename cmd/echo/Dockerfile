FROM golang:1.25.5 AS builder

WORKDIR /

FROM builder AS deps

COPY go.mod go.sum ./

FROM deps AS build

COPY . .

ENV GOOS=linux
ENV CGO_ENABLED=0

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  go build -trimpath -ldflags="-s -w" -o echo ./cmd/echo

FROM gcr.io/distroless/static-debian12:nonroot AS runtime

COPY --from=build /echo /echo

EXPOSE 15000

ENTRYPOINT [ "/echo" ]
