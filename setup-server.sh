#!/usr/bin/env sh

# Compatibility entry point; the server profile now lives in the Go CLI.
set -eu
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
exec "$script_dir/setup.sh" --profile server "$@"
