#!/usr/bin/env bash
# gen-dev-key.sh — generate a self-signed GlassOS dev CA certificate.
#
# The private key is written to dev-ca.key.pem (never committed).
# The certificate is written to dev-ca.pem (committed for dev builds).
#
# Usage: ./gen-dev-key.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KEY="${SCRIPT_DIR}/dev-ca.key.pem"
CERT="${SCRIPT_DIR}/dev-ca.pem"

openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
    -keyout "${KEY}" \
    -out "${CERT}" \
    -subj "/CN=GlassOS RAUC Dev CA" \
    -extensions v3_ca \
    -addext "basicConstraints=critical,CA:true" \
    -addext "subjectKeyIdentifier=hash" \
    -addext "authorityKeyIdentifier=keyid:always"

echo "Generated:"
echo "  key:  ${KEY}  (keep secret — do not commit)"
echo "  cert: ${CERT} (commit this)"

