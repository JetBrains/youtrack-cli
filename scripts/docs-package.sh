#!/bin/sh
# Packages docs/ as a Writerside tree for the YouTrack Developer Portal (youtrack-dev-portal-docs).
# The portal merges the archive with importTree: youtrack-cli.tree and topics/ at the archive root.
set -eu

out=${1:-dist/youtrack-cli-docs}
rm -rf "$out" "$out.zip"
mkdir -p "$out"
cp docs/youtrack-cli.tree "$out/"
cp -R docs/topics "$out/"
(cd "$out" && zip -qr "../${out##*/}.zip" youtrack-cli.tree topics)
echo "$out.zip"
