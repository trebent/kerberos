#! /bin/bash

set -euo pipefail

REPO="trebent/kerberos"

echo "Fetching all cache IDs for ${REPO}..."

gh cache list --repo "$REPO" --limit 100 --json id --jq '.[].id' | while read -r id; do
  echo "Deleting cache $id..."
  gh cache delete "$id" --repo "$REPO"
done

echo "Done."
