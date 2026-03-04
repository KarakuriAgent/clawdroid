#!/usr/bin/env bash
# setup-cloud-env.sh — Claude Code on the Web: cloud build environment setup
# Called by .claude/settings.json SessionStart hook.
# On non-cloud environments ($CLAUDE_CODE_REMOTE != "true"), exits immediately.
set -euo pipefail

# ── 1. Cloud detection ──────────────────────────────────────────────
if [[ "${CLAUDE_CODE_REMOTE:-}" != "true" ]]; then
  exit 0
fi

PROJECT_DIR="${CLAUDE_PROJECT_DIR:-.}"
ENV_FILE="${CLAUDE_ENV_FILE:-}"

echo "[setup-cloud-env] Cloud environment detected. Setting up build tools..."

# ── 2. Go installation ──────────────────────────────────────────────
# Prints the GOROOT path to stdout. All log messages go to stderr.
install_go() {
  local required_version
  required_version=$(grep '^go ' "$PROJECT_DIR/go.mod" | awk '{print $2}')
  if [[ -z "$required_version" ]]; then
    echo "[setup-cloud-env] ERROR: Could not read Go version from go.mod" >&2
    return 1
  fi

  local go_root="$HOME/.local/go/go${required_version}"
  local go_bin="$go_root/bin/go"

  # Check if correct version is already installed
  if [[ -x "$go_bin" ]]; then
    local installed_version
    installed_version=$("$go_bin" version | awk '{print $3}' | sed 's/^go//')
    if [[ "$installed_version" == "$required_version" ]]; then
      echo "[setup-cloud-env] Go $required_version already installed. Skipping." >&2
      echo "$go_root"
      return 0
    fi
  fi

  # Determine architecture
  local arch
  arch=$(uname -m)
  case "$arch" in
    x86_64)  arch="amd64" ;;
    aarch64) arch="arm64" ;;
    armv7l)  arch="armv6l" ;;
  esac

  local tarball="go${required_version}.linux-${arch}.tar.gz"
  local url="https://go.dev/dl/${tarball}"

  echo "[setup-cloud-env] Installing Go $required_version (linux/$arch)..." >&2
  mkdir -p "$HOME/.local/go"

  local tmp_dir
  tmp_dir=$(mktemp -d)
  trap "rm -rf '$tmp_dir'" RETURN

  curl -fsSL "$url" -o "$tmp_dir/$tarball"
  rm -rf "$go_root"
  mkdir -p "$go_root"
  tar -xzf "$tmp_dir/$tarball" -C "$go_root" --strip-components=1

  echo "[setup-cloud-env] Go $required_version installed to $go_root" >&2
  echo "$go_root"
}

GO_ROOT=$(install_go)

# ── 3. Android SDK installation ─────────────────────────────────────
install_android_sdk() {
  local sdk_home="$HOME/.android-sdk"
  local cmdline_tools_dir="$sdk_home/cmdline-tools/latest"
  local sdkmanager="$cmdline_tools_dir/bin/sdkmanager"

  local components=(
    "platform-tools"
    "platforms;android-36"
    "build-tools;36.0.0"
  )

  # Check if SDK and all components are already installed
  if [[ -x "$sdkmanager" ]]; then
    local all_installed=true
    local installed_list
    installed_list=$("$sdkmanager" --list_installed 2>/dev/null || true)
    for comp in "${components[@]}"; do
      if ! echo "$installed_list" | grep -q "$comp"; then
        all_installed=false
        break
      fi
    done
    if $all_installed; then
      echo "[setup-cloud-env] Android SDK already installed. Skipping."
      return 0
    fi
  fi

  echo "[setup-cloud-env] Installing Android SDK cmdline-tools..."
  mkdir -p "$sdk_home"

  # Download cmdline-tools if not present
  if [[ ! -x "$sdkmanager" ]]; then
    local cmdline_zip="commandlinetools-linux-11076708_latest.zip"
    local cmdline_url="https://dl.google.com/android/repository/${cmdline_zip}"

    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf '$tmp_dir'" RETURN

    curl -fsSL "$cmdline_url" -o "$tmp_dir/$cmdline_zip"
    unzip -q "$tmp_dir/$cmdline_zip" -d "$tmp_dir"

    mkdir -p "$cmdline_tools_dir"
    # The zip extracts to cmdline-tools/; move contents to latest/
    cp -r "$tmp_dir/cmdline-tools/"* "$cmdline_tools_dir/"
  fi

  # Accept licenses and install components
  echo "[setup-cloud-env] Installing SDK components: ${components[*]}"
  yes | "$sdkmanager" --licenses >/dev/null 2>&1 || true
  "$sdkmanager" "${components[@]}"

  # Generate android/local.properties
  local local_props="$PROJECT_DIR/android/local.properties"
  if [[ -d "$PROJECT_DIR/android" ]]; then
    echo "sdk.dir=$sdk_home" > "$local_props"
    echo "[setup-cloud-env] Generated $local_props"
  fi

  echo "[setup-cloud-env] Android SDK installed to $sdk_home"
}

install_android_sdk
ANDROID_HOME="$HOME/.android-sdk"

# ── 4. Java 17 verification ─────────────────────────────────────────
check_java() {
  if command -v java >/dev/null 2>&1; then
    local java_version
    java_version=$(java -version 2>&1 | head -1)
    echo "[setup-cloud-env] Java found: $java_version"
  else
    echo "[setup-cloud-env] WARNING: Java not found. Android builds will fail."
    echo "[setup-cloud-env] Install Java 17+: apt install openjdk-17-jdk"
  fi
}

check_java

# Detect JAVA_HOME
JAVA_HOME_DETECTED=""
if [[ -n "${JAVA_HOME:-}" ]]; then
  JAVA_HOME_DETECTED="$JAVA_HOME"
elif command -v java >/dev/null 2>&1; then
  # Resolve symlinks to find the real java binary, then derive JAVA_HOME
  local_java_bin=$(readlink -f "$(command -v java)" 2>/dev/null || true)
  if [[ -n "$local_java_bin" ]]; then
    # java binary is typically at $JAVA_HOME/bin/java
    JAVA_HOME_DETECTED=$(dirname "$(dirname "$local_java_bin")")
  fi
fi

# ── 5. Write environment variables to $CLAUDE_ENV_FILE ───────────────
if [[ -n "$ENV_FILE" ]]; then
  {
    echo "export GOROOT=\"$GO_ROOT\""
    echo "export PATH=\"$GO_ROOT/bin:\$HOME/go/bin:$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools:\$PATH\""
    echo "export GOTOOLCHAIN=local"
    echo "export ANDROID_HOME=\"$ANDROID_HOME\""
    echo "export ANDROID_SDK_ROOT=\"$ANDROID_HOME\""
    if [[ -n "$JAVA_HOME_DETECTED" ]]; then
      echo "export JAVA_HOME=\"$JAVA_HOME_DETECTED\""
    fi
  } > "$ENV_FILE"
  echo "[setup-cloud-env] Environment variables written to $ENV_FILE"
else
  echo "[setup-cloud-env] WARNING: CLAUDE_ENV_FILE not set. Environment variables not persisted."
fi

echo "[setup-cloud-env] Setup complete."
