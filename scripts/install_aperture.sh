#!/usr/bin/env bash
set -euo pipefail
echo "Installing prerequisites (Node, Go)..."
if ! command -v brew >/dev/null 2>&1; then
  echo "Homebrew not found. Install from https://brew.sh and re-run."
  exit 1
fi
brew install node go || true
echo "Cloning and installing Aperture..."
if [ ! -d "$HOME/src" ]; then mkdir -p "$HOME/src"; fi
cd "$HOME/src"
if [ ! -d "aperture" ]; then
  git clone https://github.com/lightninglabs/aperture.git
fi
cd aperture
make install
echo "Done. Next: set LNC env vars and run 'aperture'."
