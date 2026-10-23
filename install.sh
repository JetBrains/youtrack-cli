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
# Fingerprint of the JetBrains key that signs checksums.txt (env.FINGERPRINT in .teamcity.yml).
SIGNING_KEY_FPR="B46DC71E03FEEB7F89D1F2491F7A8F87B9D8F501"
TMP_DIR=""
STAGED_BIN=""

_POSITIONAL=()
for _arg in "$@"; do
    case "$_arg" in
        *) _POSITIONAL+=("$_arg") ;;
    esac
done
RELEASE="${_POSITIONAL[0]:-}"
OUT_DIR="${_POSITIONAL[1]:-}"
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

choose_install_dir() {
    [[ -z "$OUT_DIR" ]] || return 0
    local existing
    existing="$(type -P "$PROG" || true)"
    if [[ -n "$existing" ]]; then
        if [[ -L "$existing" ]]; then
            fail "existing $PROG is a symlink ($existing); update through its original install method or specify a destination explicitly"
        fi
        case "$existing" in
            */Cellar/*|*/homebrew/*|*/node_modules/*|*/go/bin/*|/usr/bin/*|/bin/*)
                fail "existing $PROG appears package-managed ($existing); update through its original install method or specify a destination explicitly" ;;
        esac
        OUT_DIR="$(dirname "$existing")"
    else
        OUT_DIR="$HOME/.local/bin"
    fi
}

configure_path() {
    case ":$PATH:" in
        *":$OUT_DIR:"*) return 0 ;;
    esac

    local path_line profile
    local profiles=()
    # Keep $PATH literal in the shell startup file.
    # shellcheck disable=SC2016
    printf -v path_line 'export PATH=%q:"$PATH"' "$OUT_DIR"
    case "${SHELL:-}" in
        */zsh) profiles=("${ZDOTDIR:-$HOME}/.zshrc") ;;
        */bash)
            profiles=("$HOME/.bashrc")
            if [[ -f "$HOME/.bash_profile" ]]; then
                profiles+=("$HOME/.bash_profile")
            elif [[ -f "$HOME/.bash_login" ]]; then
                profiles+=("$HOME/.bash_login")
            else
                profiles+=("$HOME/.profile")
            fi ;;
        *) printf '\nAdd %s to your shell PATH to run %s.\n' "$OUT_DIR" "$PROG"; return 0 ;;
    esac
    for profile in "${profiles[@]}"; do
        if ! grep -Fqx -- "$path_line" "$profile" 2>/dev/null; then
            if mkdir -p "$(dirname "$profile")" && printf '\n%s\n' "$path_line" >> "$profile"; then
                printf 'Added %s to PATH in %s\n' "$OUT_DIR" "$profile"
            else
                printf 'Could not update %s; add the PATH entry manually.\n' "$profile" >&2
            fi
        fi
    done
    printf '\nRestart your shell, or update PATH in your current shell:\n  %s\n\n' "$path_line"
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

download_to_file() {
    local url="$1" destination="$2"
    if command -v curl >/dev/null 2>&1; then
        curl --proto '=https' --tlsv1.2 --fail --location --silent --show-error "$url" --output "$destination"
    elif command -v wget >/dev/null 2>&1; then
        # GNU wget can refuse non-https redirects and old TLS; BusyBox wget has no such options, and the checksum and signature checks still apply.
        if [[ "$(wget --help 2>&1 || true)" == *--https-only* ]]; then
            wget --https-only --secure-protocol=TLSv1_2 -qO "$destination" "$url"
        else
            printf 'Note: this wget cannot restrict downloads to https; install curl for that.\n' >&2
            wget -qO "$destination" "$url"
        fi
    else
        fail "neither curl nor wget is installed"
    fi
}

verify_checksum() {
    local archive="$1" checksums="$2" expected actual
    expected="$(awk -v name="$archive" '$2 == name || $2 == "*" name { print $1; exit }' "$checksums" | tr '[:upper:]' '[:lower:]')"
    [[ "$expected" =~ ^[0-9a-fA-F]{64}$ ]] || fail "checksum entry not found for $archive"

    if command -v sha256sum >/dev/null 2>&1; then
        actual="$(sha256sum "$archive" | awk '{print $1}' | tr '[:upper:]' '[:lower:]')"
    elif command -v shasum >/dev/null 2>&1; then
        actual="$(shasum -a 256 "$archive" | awk '{print $1}' | tr '[:upper:]' '[:lower:]')"
    else
        fail "sha256sum or shasum is required to verify the download"
    fi
    [[ "$actual" == "$expected" ]] || fail "checksum verification failed for $archive"
}

# signing_key prints the armored public key with fingerprint SIGNING_KEY_FPR ("Download <download@jetbrains.com>",
# valid until 2029-03-15). It is embedded so the check needs no key server; the VALIDSIG check below still pins the fingerprint.
signing_key() {
    cat <<'SIGNING_KEY'
-----BEGIN PGP PUBLIC KEY BLOCK-----

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
=IaXJ
-----END PGP PUBLIC KEY BLOCK-----
SIGNING_KEY
}

# verify_signature checks checksums.txt.asc against the pinned signing key, so a replaced checksums.txt is rejected.
verify_signature() {
    local checksums="$1" gnupg_home
    if ! command -v gpg >/dev/null 2>&1; then
        printf 'Note: gpg not found, checksums.txt signature not checked.\n' >&2
        return 0
    fi
    # Builds without SIGN=true publish an empty checksums.txt.asc (see .goreleaser.yaml).
    [[ -s "${checksums}.asc" ]] || fail "${checksums}.asc is empty: this release is not signed"
    gnupg_home="$(mktemp -d "${TMP_DIR}/gnupg.XXXXXX")"
    signing_key | gpg --homedir "$gnupg_home" --batch --quiet --import 2>/dev/null ||
        fail "failed to import the release signing key ${SIGNING_KEY_FPR}"
    # VALIDSIG lists the signing key fingerprint first and the primary key fingerprint last.
    gpg --homedir "$gnupg_home" --batch --status-fd 1 --verify "${checksums}.asc" "$checksums" 2>/dev/null |
        awk -v fpr="$SIGNING_KEY_FPR" '$2 == "VALIDSIG" && ($3 == fpr || $NF == fpr) { ok = 1 } END { exit !ok }' ||
        fail "signature verification failed for $checksums (expected key ${SIGNING_KEY_FPR})"
}

resolve_latest() {
    # Prefer gh CLI — works with private/internal repos
    if command -v gh >/dev/null 2>&1; then
        local tag
        tag="$(gh api "repos/${REPO}/releases/latest" --jq '.tag_name' 2>/dev/null || true)"
        if [[ -n "$tag" ]]; then
            printf '%s' "$tag"
            return
        fi
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
        fail "failed to resolve latest release from GitHub"
    printf '%s' "$tag"
}

install_youtrack() {
    [[ -n "${BASH_VERSION:-}" ]] || fail "please run with bash"

    need_cmd tar
    need_cmd find
    need_cmd cp
    need_cmd chmod
    need_cmd mv
    need_cmd mkdir
    need_cmd mktemp
    need_cmd uname

    if [[ -z "$RELEASE" ]]; then
        RELEASE="$(resolve_latest)"
    fi

    [[ "$RELEASE" =~ ^v?[0-9A-Za-z._-]+$ ]] || fail "invalid release: $RELEASE"

    local os arch version release_url target
    os="$(detect_os)"
    arch="$(detect_arch)"
    version="${RELEASE#v}"
    release_url="https://github.com/${REPO}/releases/download/${RELEASE}"
    target="${OUT_DIR}/${PROG}"

    printf '\033[0;90mInstalling %s %s for %s/%s...\033[0m\n\n' "$PROG" "$RELEASE" "$os" "$arch"

    mkdir -p "$OUT_DIR" || fail "failed to create $OUT_DIR"
    [[ -d "$OUT_DIR" ]] || fail "output path is not a directory: $OUT_DIR"
    [[ -w "$OUT_DIR" ]] || fail "$OUT_DIR is not writable; choose a writable directory such as ~/.local/bin"
    OUT_DIR="$(cd "$OUT_DIR" && pwd -L)"
    target="${OUT_DIR}/${PROG}"

    TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/${PROG}.XXXXXX")"
    cd "$TMP_DIR"

    local archive="${PROG}_${version}_${os}_${arch}.tar.gz" downloaded=""
    if command -v gh >/dev/null 2>&1 &&
        gh release download "${RELEASE}" --repo "${REPO}" --pattern "${archive}" --output "$archive" 2>/dev/null &&
        gh release download "${RELEASE}" --repo "${REPO}" --pattern checksums.txt --output checksums.txt 2>/dev/null &&
        gh release download "${RELEASE}" --repo "${REPO}" --pattern checksums.txt.asc --output checksums.txt.asc 2>/dev/null; then
        downloaded=yes
    fi
    if [[ -z "$downloaded" ]]; then
        download_to_file "${release_url}/${archive}" "$archive"
        download_to_file "${release_url}/checksums.txt" checksums.txt
        download_to_file "${release_url}/checksums.txt.asc" checksums.txt.asc
    fi
    verify_signature checksums.txt
    verify_checksum "$archive" checksums.txt
    tar -xzf "$archive" || fail "failed to extract $archive"

    local tmp_bin
    tmp_bin="$(find . -type f -name "$PROG" -print -quit)"
    [[ -n "$tmp_bin" ]] || fail "binary not found in archive"

    STAGED_BIN="$(mktemp "${OUT_DIR}/.${PROG}.XXXXXX")"
    cp "$tmp_bin" "$STAGED_BIN"
    chmod 0755 "$STAGED_BIN"
    mv -f "$STAGED_BIN" "$target"
    STAGED_BIN=""

    printf '%s Installed %s at %s\n\n' "$CHECK_MARK" "$PROG" "$target"
    "$target" --version || fail "installed binary failed to run"

    configure_path

    # 'youtrack update' sets YOUTRACK_INSTALL_UPDATE: the user already knows the next steps.
    if [[ -n "${YOUTRACK_INSTALL_UPDATE:-}" ]]; then
        return 0
    fi
    printf '\n\033[1mNext steps:\033[0m\n\n'
    printf '  Authenticate with your YouTrack instance:\n'
    printf '    \033[0;90myoutrack instance add\033[0m\n\n'
    printf '  Verify connection:\n'
    printf '    \033[0;90myoutrack user current\033[0m\n\n'
}

trap cleanup EXIT
trap 'fail "interrupted"' INT TERM

choose_install_dir

if [[ -z "${YOUTRACK_INSTALL_UPDATE:-}" ]]; then
cat << 'LOGO'

 __   __         _____               _
 \ \ / /__  _   |_   _| __ __ _  ___| | __
  \ V / _ \| | | || || '__/ _`|/ __| |/ /
   | | (_) | |_| || || | | (_| | (__|   <
   |_|\___/ \__,_||_||_|  \__,_|\___|_|\_\

LOGO

printf 'This script installs \033[1myoutrack\033[0m to %s.\n' "$OUT_DIR"
printf 'New installs use ~/.local/bin without sudo; existing unmanaged installs are updated in place.\n'
printf 'Install to a different directory:  \033[0;90mcurl -fsSL https://jb.gg/youtrack-cli/install | bash -s -- "" ~/.local/bin\033[0m\n'
printf 'Install a specific version:        \033[0;90mcurl -fsSL https://jb.gg/youtrack-cli/install | bash -s -- v0.5.1\033[0m\n\n'
fi

install_youtrack
