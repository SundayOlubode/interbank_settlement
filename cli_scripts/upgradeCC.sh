## Upgrade chaincode

BASE_PATH="/opt/gopath/src/github.com/hyperledger/fabric/peer"

set -e # Exit on error

# Track Chaincode Version
CHAINCODE_NAME="account"
CHAINCODE_LANG="golang"
CHAINCODE_PATH="$BASE_PATH/contracts"
CHAINCODE_LABEL_BASE="account"

PACKAGE_DIR="$BASE_PATH/cc-packages"
VERSION_FILE="$BASE_PATH/cc-version.txt"

mkdir -p "$PACKAGE_DIR"

# === Determine new version ===
if [ -f "$VERSION_FILE" ]; then
  CURRENT_VERSION=$(cat "$VERSION_FILE")
else
  CURRENT_VERSION="1.0"
fi

IFS='.' read -r MAJOR MINOR <<< "$CURRENT_VERSION"
NEW_MINOR=$((MINOR + 1))
NEW_VERSION="${MAJOR}.${NEW_MINOR}"
NEW_LABEL="${CHAINCODE_LABEL_BASE}_${NEW_VERSION}"
PACKAGE_NAME="${NEW_LABEL}.tar.gz"

# === Save new version ===
echo "$NEW_VERSION" > "$VERSION_FILE"
echo "✅ New CC version saved: $NEW_VERSION"
