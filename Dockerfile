FROM golang:1.25.5 AS builder

WORKDIR /

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
  go mod download

COPY cmd/ cmd/
COPY internal/ internal/

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o kerberos ./cmd/kerberos

FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /

COPY --from=builder /kerberos /kerberos

ARG VERSION="unset"

ENV VERSION=${VERSION}

EXPOSE 30000

ENTRYPOINT [ "/kerberos" ]
