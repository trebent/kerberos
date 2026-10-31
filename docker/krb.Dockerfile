FROM golang:1.27.3@sha256:2920bb124f1a2e12a1d39f0cab756ae0068441ee1da7655dec7f5e374944095b AS builder

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

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab AS runtime

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

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab AS poc-runtime

USER nonroot:nonroot

COPY --chown=nonroot:nonroot --from=build /kerberos /kerberos
COPY --chown=nonroot:nonroot --from=build openapi/ /krb-oas/
COPY --chown=nonroot:nonroot --from=build poc/poc.json /poc.json

ARG VERSION="unset"

ENV VERSION=${VERSION}
ENV OAS_DIRECTORY=/krb-oas

EXPOSE 30000

ENTRYPOINT [ "/kerberos" ]
