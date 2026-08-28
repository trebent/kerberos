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
  go build -trimpath -ldflags="-s -w" -o kerberos .

FROM gcr.io/distroless/static-debian12:nonroot@sha256:1b7b9f0f0e0a1d2155f531db587cc48ec26aaf97ab64364225f5bf18a054e66a AS runtime

USER nonroot:nonroot

COPY --chown=nonroot:nonroot --from=build /kerberos /kerberos
COPY --chown=nonroot:nonroot --from=build openapi/ /krb-oas/

ARG VERSION="unset"

ENV VERSION=${VERSION}
ENV OAS_DIRECTORY=/krb-oas

EXPOSE 30000

ENTRYPOINT [ "/kerberos" ]

#
# PoC-runtime image for AWS ECR. This image is used to run the PoC in AWS ECR, and as such has the PoC JSON file baked in.
#

FROM gcr.io/distroless/static-debian12:nonroot@sha256:1b7b9f0f0e0a1d2155f531db587cc48ec26aaf97ab64364225f5bf18a054e66a AS poc-runtime

USER nonroot:nonroot

COPY --chown=nonroot:nonroot --from=build /kerberos /kerberos
COPY --chown=nonroot:nonroot --from=build openapi/ /krb-oas/
COPY --chown=nonroot:nonroot --from=build poc/poc.json /poc.json

ARG VERSION="unset"

ENV VERSION=${VERSION}
ENV OAS_DIRECTORY=/krb-oas

EXPOSE 30000

ENTRYPOINT [ "/kerberos" ]
