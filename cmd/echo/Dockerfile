FROM golang:1.24.4 AS builder

WORKDIR /

COPY go.mod go.sum ./
COPY cmd/ cmd/
COPY internal/ internal/

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o echo ./cmd/echo

FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /

COPY --from=builder /echo /echo

EXPOSE 15000

ENTRYPOINT [ "/echo" ]
