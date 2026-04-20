#!/usr/bin/env bash
# =============================================================================
# WHOOP Stats — Interactive Setup Wizard
# =============================================================================
# Bootstraps a fresh whoop-stats deployment in minutes.
#
# What it does:
#   1. Creates .env from .env.example (if needed)
#   2. Auto-generates cryptographic secrets (ENCRYPTION_KEY, POSTGRES_PASSWORD)
#   3. Prompts for WHOOP API credentials (client ID + secret)
#   4. Runs the OAuth flow to generate .whoop_token.json
#   5. Auto-detects your WHOOP User ID from the token
#   6. Validates everything is ready for deployment
#
# Usage:
#   ./setup.sh              # Full interactive setup
#   ./setup.sh --validate   # Validate existing .env only (no prompts)
#
# Safe to run multiple times — skips steps that are already complete.
# =============================================================================

set -euo pipefail

# --- Colors & formatting ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m' # No Color

info()    { echo -e "${BLUE}ℹ${NC}  $*"; }
success() { echo -e "${GREEN}✔${NC}  $*"; }
warn()    { echo -e "${YELLOW}⚠${NC}  $*"; }
error()   { echo -e "${RED}✖${NC}  $*"; }
header()  { echo -e "\n${BOLD}${CYAN}$*${NC}"; }
dim()     { echo -e "${DIM}$*${NC}"; }

ENV_FILE=".env"
ENV_EXAMPLE=".env.example"
TOKEN_FILE=".whoop_token.json"
VALIDATE_ONLY=false

# --- Parse arguments ---
for arg in "$@"; do
    case $arg in
        --validate) VALIDATE_ONLY=true ;;
        --help|-h)
            echo "Usage: ./setup.sh [--validate]"
            echo ""
            echo "Options:"
            echo "  --validate   Validate existing .env only (no prompts)"
            echo "  --help       Show this help message"
            exit 0
            ;;
    esac
done

# --- Utility functions ---

# Generate a cryptographically secure random hex string of N bytes.
generate_hex() {
    local bytes="${1:-16}"
    if command -v openssl &>/dev/null; then
        openssl rand -hex "$bytes"
    elif [[ -f /dev/urandom ]]; then
        head -c "$bytes" /dev/urandom | od -An -tx1 | tr -d ' \n'
    else
        error "Cannot generate random bytes — install openssl or ensure /dev/urandom exists."
        exit 1
    fi
}

# Read a variable's value from .env (handles comments and whitespace).
get_env_value() {
    local key="$1"
    if [[ ! -f "$ENV_FILE" ]]; then
        echo ""
        return
    fi
    grep -E "^${key}=" "$ENV_FILE" 2>/dev/null | head -1 | cut -d'=' -f2- | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' || echo ""
}

# Set a variable in .env. If it exists, replace it. If not, append it.
set_env_value() {
    local key="$1"
    local value="$2"
    if grep -qE "^${key}=" "$ENV_FILE" 2>/dev/null; then
        # Replace existing value (macOS + Linux compatible sed)
        if [[ "$(uname)" == "Darwin" ]]; then
            sed -i '' "s|^${key}=.*|${key}=${value}|" "$ENV_FILE"
        else
            sed -i "s|^${key}=.*|${key}=${value}|" "$ENV_FILE"
        fi
    else
        echo "${key}=${value}" >> "$ENV_FILE"
    fi
}

# Prompt the user for a value, with an optional default.
prompt_value() {
    local prompt_text="$1"
    local default_value="${2:-}"
    local value=""

    if [[ -n "$default_value" ]]; then
        read -rp "$(echo -e "${BLUE}?${NC}  ${prompt_text} [${default_value}]: ")" value
        echo "${value:-$default_value}"
    else
        while [[ -z "$value" ]]; do
            read -rp "$(echo -e "${BLUE}?${NC}  ${prompt_text}: ")" value
            if [[ -z "$value" ]]; then
                warn "This value is required."
            fi
        done
        echo "$value"
    fi
}

# =============================================================================
# Step 1: Check prerequisites
# =============================================================================
check_prerequisites() {
    header "Step 1/6: Checking prerequisites"
    
    local missing=0

    if command -v docker &>/dev/null; then
        success "Docker $(docker --version | awk '{print $3}' | tr -d ',')"
    else
        error "Docker is not installed. Install it from https://docs.docker.com/get-docker/"
        missing=1
    fi

    if command -v docker compose &>/dev/null || command -v docker-compose &>/dev/null; then
        success "Docker Compose available"
    else
        error "Docker Compose is not available."
        missing=1
    fi

    if command -v go &>/dev/null; then
        success "Go $(go version | awk '{print $3}' | sed 's/go//')"
    else
        warn "Go is not installed. Required for the one-time OAuth token generation."
        warn "Install from https://go.dev/dl/ or skip if you already have .whoop_token.json"
    fi

    if [[ ! -f "$ENV_EXAMPLE" ]]; then
        error ".env.example not found. Are you in the whoop-stats project root?"
        exit 1
    fi

    if [[ "$missing" -eq 1 ]]; then
        error "Missing required prerequisites. Please install them and re-run."
        exit 1
    fi
}

# =============================================================================
# Step 2: Create .env
# =============================================================================
create_env() {
    header "Step 2/6: Environment file"

    if [[ -f "$ENV_FILE" ]]; then
        success ".env already exists — will update missing values"
    else
        cp "$ENV_EXAMPLE" "$ENV_FILE"
        success "Created .env from .env.example"
    fi
}

# =============================================================================
# Step 3: Generate secrets
# =============================================================================
generate_secrets() {
    header "Step 3/6: Generating secrets"

    # --- Encryption Key ---
    local enc_key
    enc_key=$(get_env_value "ENCRYPTION_KEY")
    if [[ -z "$enc_key" ]]; then
        enc_key=$(generate_hex 16)
        set_env_value "ENCRYPTION_KEY" "$enc_key"
        success "Generated ENCRYPTION_KEY (32-char AES-256 key)"
    elif [[ ${#enc_key} -ne 32 ]]; then
        warn "ENCRYPTION_KEY is ${#enc_key} chars (must be 32). Regenerating..."
        enc_key=$(generate_hex 16)
        set_env_value "ENCRYPTION_KEY" "$enc_key"
        success "Regenerated ENCRYPTION_KEY (32-char AES-256 key)"
    else
        success "ENCRYPTION_KEY already set (${#enc_key} chars)"
    fi

    # --- Postgres Password ---
    local pg_pass
    pg_pass=$(get_env_value "POSTGRES_PASSWORD")
    if [[ -z "$pg_pass" || "$pg_pass" == "changeme" ]]; then
        pg_pass=$(generate_hex 16)
        set_env_value "POSTGRES_PASSWORD" "$pg_pass"
        success "Generated POSTGRES_PASSWORD"
    else
        success "POSTGRES_PASSWORD already set"
    fi
}

# =============================================================================
# Step 4: WHOOP API credentials
# =============================================================================
configure_whoop_credentials() {
    header "Step 4/6: WHOOP API credentials"

    local client_id
    client_id=$(get_env_value "WHOOP_CLIENT_ID")
    local client_secret
    client_secret=$(get_env_value "WHOOP_CLIENT_SECRET")

    if [[ -n "$client_id" && -n "$client_secret" ]]; then
        success "WHOOP_CLIENT_ID already set"
        success "WHOOP_CLIENT_SECRET already set"
        return
    fi

    echo ""
    info "You need a WHOOP Developer App to continue."
    dim "  1. Go to https://developer.whoop.com"
    dim "  2. Create a new application"
    dim "  3. Add redirect URI: http://localhost:8081/callback"
    dim "  4. Copy your Client ID and Client Secret"
    echo ""

    if [[ -z "$client_id" ]]; then
        client_id=$(prompt_value "WHOOP Client ID")
        set_env_value "WHOOP_CLIENT_ID" "$client_id"
        success "Saved WHOOP_CLIENT_ID"
    else
        success "WHOOP_CLIENT_ID already set"
    fi

    if [[ -z "$client_secret" ]]; then
        client_secret=$(prompt_value "WHOOP Client Secret")
        set_env_value "WHOOP_CLIENT_SECRET" "$client_secret"
        success "Saved WHOOP_CLIENT_SECRET"
    else
        success "WHOOP_CLIENT_SECRET already set"
    fi
}

# =============================================================================
# Step 5: OAuth token generation
# =============================================================================
generate_oauth_token() {
    header "Step 5/6: OAuth token generation"

    if [[ -f "$TOKEN_FILE" ]]; then
        success ".whoop_token.json already exists"
        dim "  To regenerate, delete .whoop_token.json and re-run setup.sh"
        
        # Still try to detect user ID if not set
        local user_id
        user_id=$(get_env_value "WHOOP_USER_ID")
        if [[ -z "$user_id" ]]; then
            warn "WHOOP_USER_ID is not set. Running auth refresh to detect it..."
            run_auth_cli
        fi
        return
    fi

    if ! command -v go &>/dev/null; then
        warn "Go is not installed — cannot run the OAuth flow."
        echo ""
        info "To complete setup, run these commands on a machine with Go + a browser:"
        dim "  export WHOOP_CLIENT_ID=$(get_env_value 'WHOOP_CLIENT_ID')"
        dim "  export WHOOP_CLIENT_SECRET=<your_secret>"
        dim "  go run cmd/auth/main.go"
        echo ""
        info "Then copy .whoop_token.json to this directory."
        return
    fi

    echo ""
    info "Starting OAuth flow — a browser window will open for WHOOP authorization."
    info "Make sure http://localhost:8081/callback is in your WHOOP App's Redirect URIs."
    echo ""

    run_auth_cli
}

run_auth_cli() {
    local client_id
    client_id=$(get_env_value "WHOOP_CLIENT_ID")
    local client_secret
    client_secret=$(get_env_value "WHOOP_CLIENT_SECRET")

    WHOOP_CLIENT_ID="$client_id" WHOOP_CLIENT_SECRET="$client_secret" \
        go run cmd/auth/main.go

    if [[ -f "$TOKEN_FILE" ]]; then
        success "OAuth tokens saved to .whoop_token.json"
    else
        warn "Token file was not created. You may need to retry the OAuth flow."
    fi
}

# =============================================================================
# Step 6: Validate
# =============================================================================
validate_env() {
    header "Step 6/6: Validating configuration"

    local errors=0

    # Required variables
    local required_vars=("ENCRYPTION_KEY" "POSTGRES_PASSWORD" "WHOOP_CLIENT_ID" "WHOOP_CLIENT_SECRET" "WHOOP_USER_ID")
    for var in "${required_vars[@]}"; do
        local val
        val=$(get_env_value "$var")
        if [[ -z "$val" ]]; then
            error "${var} is not set"
            errors=$((errors + 1))
        else
            success "${var} ✓"
        fi
    done

    # Validate ENCRYPTION_KEY length
    local enc_key
    enc_key=$(get_env_value "ENCRYPTION_KEY")
    if [[ -n "$enc_key" && ${#enc_key} -ne 32 ]]; then
        error "ENCRYPTION_KEY must be exactly 32 characters (got ${#enc_key})"
        errors=$((errors + 1))
    fi

    # Check token file
    if [[ -f "$TOKEN_FILE" ]]; then
        success ".whoop_token.json ✓"
    else
        warn ".whoop_token.json not found — run 'go run cmd/auth/main.go' to generate"
        errors=$((errors + 1))
    fi

    echo ""
    if [[ "$errors" -gt 0 ]]; then
        error "${errors} issue(s) found. Fix them and re-run: ./setup.sh --validate"
        return 1
    else
        success "All checks passed!"
        return 0
    fi
}

# =============================================================================
# Summary
# =============================================================================
print_summary() {
    echo ""
    header "══════════════════════════════════════════"
    header "  🎉 whoop-stats is ready to deploy!"
    header "══════════════════════════════════════════"
    echo ""
    info "Start the stack:"
    echo ""
    dim "  # Homelab / NAS (polling mode, recommended)"
    echo -e "  ${BOLD}docker compose up -d --build${NC}"
    echo ""
    dim "  # Production (with named volumes)"
    echo -e "  ${BOLD}docker compose -f docker-compose.prod.yml up -d --build${NC}"
    echo ""
    info "Dashboard:  ${BOLD}http://localhost:$(get_env_value 'FRONTEND_PORT' || echo '3032')${NC}"
    info "API:        ${BOLD}http://localhost:$(get_env_value 'BACKEND_PORT' || echo '8085')${NC}"
    echo ""
    dim "Run './setup.sh --validate' at any time to check your configuration."
    echo ""
}

# =============================================================================
# Main
# =============================================================================
main() {
    echo ""
    echo -e "${BOLD}${CYAN}╔══════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}${CYAN}║     WHOOP Stats — Setup Wizard           ║${NC}"
    echo -e "${BOLD}${CYAN}╚══════════════════════════════════════════╝${NC}"
    echo ""

    if [[ "$VALIDATE_ONLY" == "true" ]]; then
        validate_env
        exit $?
    fi

    check_prerequisites
    create_env
    generate_secrets
    configure_whoop_credentials
    generate_oauth_token
    
    if validate_env; then
        print_summary
    fi
}

main "$@"
