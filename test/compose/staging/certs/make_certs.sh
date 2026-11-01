#! /bin/bash
set -euo pipefail

if ! command -v mkcert &>/dev/null; then
  echo "Error: mkcert is not installed. Please install it first: https://github.com/FiloSottile/mkcert" >&2
  exit 1
fi

mkcert -install

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

mkcert \
  -cert-file "$SCRIPT_DIR/krb.pem" \
  -key-file  "$SCRIPT_DIR/krb-key.pem" \
  gw.trebent.test admin.trebent.test localhost

mkcert \
  -cert-file "$SCRIPT_DIR/connector-jaeger.pem" \
  -key-file  "$SCRIPT_DIR/connector-jaeger-key.pem" \
  jaeger.trebent.test localhost

mkcert \
  -cert-file "$SCRIPT_DIR/grafana.pem" \
  -key-file  "$SCRIPT_DIR/grafana-key.pem" \
  grafana.trebent.test localhost

mkcert \
  -cert-file "$SCRIPT_DIR/jaeger.pem" \
  -key-file  "$SCRIPT_DIR/jaeger-key.pem" \
  jaeger localhost

CAROOT="$(mkcert -CAROOT)"
cp "$CAROOT/rootCA.pem" "$SCRIPT_DIR/rootCA.pem"

echo "Certificates generated in $SCRIPT_DIR"

chmod a+r $SCRIPT_DIR/*
