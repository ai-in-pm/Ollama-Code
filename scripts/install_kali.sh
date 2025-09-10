#!/usr/bin/env bash
set -euo pipefail

APP_NAME="ollama-code"
INSTALL_BIN="/usr/local/bin/${APP_NAME}"
SYMLINK_BIN="/usr/bin/olc"
BUILD_DIR="$(dirname "$0")/../dist"
PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

banner() {
  echo "=========================================="
  echo "$1"
  echo "=========================================="
}

check_kali() {
  if [[ -f /etc/os-release ]]; then
    . /etc/os-release
    if [[ "${ID:-}" != "kali" ]]; then
      echo "Warning: This installer targets Kali Linux. Detected: ${PRETTY_NAME:-unknown}."
      read -rp "Continue anyway? [y/N] " ans
      [[ "${ans}" == "y" || "${ans}" == "Y" ]] || exit 1
    fi
  fi
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Installing missing dependency: $1"
    sudo apt-get update -y
    case "$1" in
      go) sudo apt-get install -y golang;;
      git) sudo apt-get install -y git;;
      curl) sudo apt-get install -y curl;;
      *) sudo apt-get install -y "$1";;
    esac
  fi
}

install_ollama() {
  if ! command -v ollama >/dev/null 2>&1; then
    banner "Installing Ollama"
    curl -fsSL https://ollama.com/install.sh | sh
  fi
}

pull_model() {
  local model="${1:-qwen2.5-coder:1.5b}"
  banner "Pulling model: ${model}"
  ollama pull "${model}" || true
}

build() {
  banner "Building ${APP_NAME}"
  mkdir -p "${BUILD_DIR}"
  (cd "${PROJECT_ROOT}" && GO111MODULE=on go mod tidy && go build -o "${BUILD_DIR}/${APP_NAME}")
}

install_bin() {
  banner "Installing binary"
  sudo install -m 0755 "${BUILD_DIR}/${APP_NAME}" "${INSTALL_BIN}"
  sudo ln -sf "${INSTALL_BIN}" "${SYMLINK_BIN}"
  echo "Installed: ${INSTALL_BIN}"
  echo "Symlink:   ${SYMLINK_BIN}"
}

post_notes() {
  echo
  echo "Try it:"
  echo "  ${APP_NAME} --help"
  echo "  olc  # shortcut"
  echo
  echo "Docs and updates: https://ollama.com/influencepm/ollama-code"
}

main() {
  banner "${APP_NAME} Kali Installer"
  check_kali
  require_cmd curl
  require_cmd git
  require_cmd go
  install_ollama
  pull_model "${OLLAMA_MODEL:-qwen2.5-coder:1.5b}"
  build
  install_bin
  post_notes
}

main "$@"

