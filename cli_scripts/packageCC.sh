## Package and install chaincode

BASE_PATH="/opt/gopath/src/github.com/hyperledger/fabric/peer"

installGo() {
  echo "\nRunning apt-get update ....\n"
  apt-get update

  echo "\n🔄 Installing wget ....\n"
  apt-get install -y wget

  wget https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz
  tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
  export PATH=$PATH:/usr/local/go/bin
  go version
}

go version
# Check if Go is installed
if [ $? -ne 0 ]; then
  echo "❌ Go is not installed." >&2

  # Install Go
  echo "🔄 Installing Go..."
  installGo

  # Check if installation was successful
  if [ $? -ne 0 ]; then
    echo "❌ Failed to install Go." >&2
    exit 1
    else
    echo "✅ Go installed successfully."
  fi
fi

set -e # Exit on error

# Track Chaincode Version
CHAINCODE_NAME="account"
CHAINCODE_LANG="golang"
CHAINCODE_PATH="$BASE_PATH/contracts"
CHAINCODE_LABEL_BASE="account"

PACKAGE_DIR="$CHAINCODE_PATH/cc-packages"
VERSION_FILE="$CHAINCODE_PATH/cc-version.txt"

mkdir -p "$PACKAGE_DIR"

# === Determine new version ===
if [ -f "$VERSION_FILE" ]; then
  CURRENT_VERSION=$(head -n 1 "$VERSION_FILE")
else
  CURRENT_VERSION="1.0"
fi

IFS='.' read -r MAJOR MINOR <<<"$CURRENT_VERSION"
NEW_MINOR=$((MINOR + 1))
NEW_VERSION="${MAJOR}.${NEW_MINOR}"
NEW_LABEL="${CHAINCODE_LABEL_BASE}_${NEW_VERSION}"
PACKAGE_NAME="${NEW_LABEL}.tar.gz"

echo "📦 Packaging chaincode as $PACKAGE_NAME (version $NEW_VERSION)..."

# === Package chaincode ===
peer lifecycle chaincode package "$PACKAGE_DIR/$PACKAGE_NAME" \
  --path "$CHAINCODE_PATH" \
  --lang "$CHAINCODE_LANG" \
  --label "$NEW_LABEL"

# === Save new version and metadata ===
{
  echo "$NEW_VERSION"
  echo "LABEL=$NEW_LABEL"
  echo "PACKAGE_PATH=$PACKAGE_DIR/$PACKAGE_NAME"
  echo "PACKAGE_NAME=$PACKAGE_NAME"
} >"$VERSION_FILE"

echo "✅ Chaincode packaged and metadata saved to $VERSION_FILE"
