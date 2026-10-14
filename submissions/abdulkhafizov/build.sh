#!/usr/bin/env bash

set -euo pipefail
cd "$(dirname "$0")"

GOOS=$(go env GOOS) GOARCH=$(go env GOARCH) go build -trimpath -o server .

