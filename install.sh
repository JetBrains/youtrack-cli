#!/usr/bin/env bash

#
# Copyright 2024 JetBrains s.r.o.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# https://www.apache.org/licenses/LICENSE-2.0
#

set -Eeuo pipefail

PROG="youtrack"
REPO="JetBrains/youtrack-cli"
CHECK_MARK=$'\033[1;32m✓\033[0m'
TMP_DIR=""
STAGED_BIN=""

_POSITIONAL=()
for _arg in "$@"; do
    case "$_arg" in
        *) _POSITIONAL+=("$_arg") ;;
    esac
done
RELEASE="${_POSITIONAL[0]:-}"
OUT_DIR="${_POSITIONAL[1]:-/usr/local/bin}"
USER_SET_OUT_DIR="${_POSITIONAL[1]:+yes}"
unset _arg _POSITIONAL

cleanup() {
    if [[ -n "${TMP_DIR:-}" && -d "$TMP_DIR" ]]; then
        rm -rf "$TMP_DIR"
    fi
    if [[ -n "${STAGED_BIN:-}" && -e "$STAGED_BIN" ]]; then
        rm -f "$STAGED_BIN"
    fi
}

fail() {
    printf '\n\033[31mError:\033[0m %s\n' "${1:-unknown error}" >&2
    exit 1
}

need_cmd() {
    command -v "$1" >/dev/null 2>&1 || fail "'$1' is required but not installed"
}

detect_os() {
    case "$(uname -s)" in
        Darwin) printf 'darwin' ;;
        Linux)  printf 'linux'  ;;
        *)      fail "unsupported OS: $(uname -s)" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   printf 'x86_64' ;;
        aarch64|arm64)  printf 'arm64'  ;;
        *)              fail "unsupported arch: $(uname -m)" ;;
    esac
}

download() {
    local url="$1"
    if command -v curl >/dev/null 2>&1; then
        curl --fail --location --silent --show-error "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$url"
    else
        fail "neither curl nor wget is installed"
    fi
}

resolve_latest() {
    # Prefer gh CLI — works with private/internal repos
    if command -v gh >/dev/null 2>&1; then
        local tag
        tag="$(gh api "repos/${REPO}/releases/latest" --jq '.tag_name' 2>/dev/null)"
        [[ -n "$tag" ]] || fail "failed to resolve latest release via gh (run: gh auth login)"
        printf '%s' "$tag"
        return
    fi

    # Public fallback: follow GitHub's /releases/latest redirect
    local location tag
    if command -v curl >/dev/null 2>&1; then
        location="$(curl -sI "https://github.com/${REPO}/releases/latest" \
            | grep -i '^location:' | head -1 | tr -d '\r')"
    else
        location="$(wget --spider -S "https://github.com/${REPO}/releases/latest" 2>&1 \
            | grep -i '^\s*location:' | tail -1 | tr -d '\r' | sed 's/^\s*//')"
    fi
    tag="${location##*/}"
    [[ -n "$tag" && "$tag" != "$location" ]] || \
        fail "failed to resolve latest release (install gh CLI and run: gh auth login)"
    printf '%s' "$tag"
}

install_youtrack() {
    [[ -n "${BASH_VERSION:-}" ]] || fail "please run with bash"

    need_cmd tar
    need_cmd chmod
    need_cmd mv
    need_cmd mkdir
    need_cmd mktemp
    need_cmd uname

    if [[ -z "$RELEASE" ]]; then
        RELEASE="$(resolve_latest)"
    fi

    [[ "$RELEASE" =~ ^v?[0-9A-Za-z._-]+$ ]] || fail "invalid release: $RELEASE"

    local os arch version url target
    os="$(detect_os)"
    arch="$(detect_arch)"
    version="${RELEASE#v}"
    url="https://github.com/${REPO}/releases/download/${RELEASE}/${PROG}_${version}_${os}_${arch}.tar.gz"
    target="${OUT_DIR}/${PROG}"

    printf '\033[0;90mInstalling %s %s for %s/%s...\033[0m\n\n' "$PROG" "$RELEASE" "$os" "$arch"

    # Fall back to ~/.local/bin if the default dir is not writable and user didn't set a dir.
    if [[ -z "$USER_SET_OUT_DIR" ]] && ! { mkdir -p "$OUT_DIR" 2>/dev/null && [[ -w "$OUT_DIR" ]]; }; then
        OUT_DIR="${HOME}/.local/bin"
        target="${OUT_DIR}/${PROG}"
        printf '\033[33mNote:\033[0m /usr/local/bin is not writable, installing to %s instead.\n' "$OUT_DIR"
        printf '      Add it to your PATH if needed: export PATH="%s:$PATH"\n\n' "$OUT_DIR"
    fi

    mkdir -p "$OUT_DIR" || fail "failed to create $OUT_DIR"
    [[ -w "$OUT_DIR" ]] || fail "$OUT_DIR is not writable
  Run with sudo:       gh release download --repo ${REPO} --pattern install.sh -O - | sudo bash
  Use writable dir:    gh release download --repo ${REPO} --pattern install.sh -O - | bash -s -- \"\" ~/.local/bin"

    TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/${PROG}.XXXXXX")"
    cd "$TMP_DIR"

    local archive="${PROG}_${version}_${os}_${arch}.tar.gz"
    if command -v gh >/dev/null 2>&1; then
        gh release download "${RELEASE}" --repo "${REPO}" --pattern "${archive}" --output - \
            | tar -xzf - || fail "gh release download failed (run: gh auth login)"
    else
        download "$url" | tar -xzf - || fail "download or extract failed from $url"
    fi

    local tmp_bin
    tmp_bin="$(find . -type f -name "$PROG" -print -quit)"
    [[ -n "$tmp_bin" ]] || fail "binary not found in archive"

    STAGED_BIN="$(mktemp "${OUT_DIR}/.${PROG}.XXXXXX")"
    cp "$tmp_bin" "$STAGED_BIN"
    chmod 0755 "$STAGED_BIN"
    mv -f "$STAGED_BIN" "$target"
    STAGED_BIN=""

    printf '%s Installed %s at %s\n\n' "$CHECK_MARK" "$PROG" "$target"
    "$target" --version

    printf '\n\033[1mNext steps:\033[0m\n\n'
    printf '  Authenticate with your YouTrack instance:\n'
    printf '    \033[0;90myoutrack instance add\033[0m\n\n'
    printf '  Verify connection:\n'
    printf '    \033[0;90myoutrack user current\033[0m\n\n'
}

trap cleanup EXIT
trap 'fail "interrupted"' INT TERM

cat << 'LOGO'

 __   __         _____               _
 \ \ / /__  _   |_   _| __ __ _  ___| | __
  \ V / _ \| | | || || '__/ _`|/ __| |/ /
   | | (_) | |_| || || | | (_| | (__|   <
   |_|\___/ \__,_||_||_|  \__,_|\___|_|\_\

LOGO

printf 'This script installs \033[1myoutrack\033[0m to your PATH.\n'
printf 'Install to a different directory:  \033[0;90mgh release download ... -O - | bash -s -- "" ~/.local/bin\033[0m\n'
printf 'Install a specific version:        \033[0;90mgh release download v1.0.0 ... -O - | bash\033[0m\n\n'

install_youtrack
