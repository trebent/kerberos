FROM golang:1.24.6 AS builder

ARG VERSION="unset"

WORKDIR /

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
  go mod download

COPY cmd/ cmd/
COPY internal/ internal/

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-X github.com/trebent/kerberos/internal/version.Ver=${VERSION} -s -w" -o kerberos ./cmd/kerberos

FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /

COPY --from=builder /kerberos /kerberos

EXPOSE 30000

ENTRYPOINT [ "/kerberos" ]
