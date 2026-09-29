#!/bin/sh
# Builds installwall, installs its shims into a throwaway home, and prints
# the PATH export needed before running "vhs demo/demo.tape". The demo
# runs a real "pip3 install" against a name that does not exist, so
# installwall's block is the only thing that stops the real pip3 from
# ever running, no cleanup needed either way.
set -eu

cd "$(dirname "$0")/.."
go build -o /tmp/installwall-demo ./cmd/installwall

export INSTALLWALL_HOME="$(mktemp -d)/.installwall"
mkdir -p "$INSTALLWALL_HOME"
"/tmp/installwall-demo" install >/dev/null
# Put installwall itself on the same PATH entry as its shims, the way a
# real install (go install, a release binary, or Homebrew) would.
cp /tmp/installwall-demo "$INSTALLWALL_HOME/bin/installwall"

echo "Run this in the shell vhs will record from:"
echo
echo "export INSTALLWALL_HOME=$INSTALLWALL_HOME"
echo "export PATH=$INSTALLWALL_HOME/bin:\$PATH"
