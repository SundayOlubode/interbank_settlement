# === Load version information ===
VERSION_FILE="/opt/gopath/src/github.com/hyperledger/fabric/peer/contracts/cc-version.txt"

if [ ! -f "$VERSION_FILE" ]; then
  echo "❌ VERSION_FILE not found at $VERSION_FILE" >&2
  exit 1
fi

# Read values from VERSION_FILE
NEW_VERSION=$(sed -n '1p' "$VERSION_FILE")
NEW_LABEL=$(grep '^LABEL=' "$VERSION_FILE" | cut -d '=' -f2)
PACKAGE_NAME=$(grep '^PACKAGE_NAME=' "$VERSION_FILE" | cut -d '=' -f2)
PACKAGE_PATH=$(grep '^PACKAGE_PATH=' "$VERSION_FILE" | cut -d '=' -f2)

# Fallback if PACKAGE_PATH is not explicitly written
if [ -z "$PACKAGE_PATH" ]; then
  PACKAGE_PATH="/opt/gopath/src/github.com/hyperledger/fabric/peer/contracts/cc-packages/$PACKAGE_NAME"
fi

# === Install chaincode ===
echo "🔄 Installing chaincode $NEW_LABEL on peer $CORE_PEER_LOCALMSPID..."
if peer lifecycle chaincode install "$PACKAGE_PATH"; then
  echo "✅ Chaincode $NEW_LABEL installed on peer $CORE_PEER_LOCALMSPID."
else
  echo "❌ Failed to install chaincode $NEW_LABEL" >&2
  exit 1
fi

# Export the package ID of the installed chaincode
export CC_PACKAGE_ID=$(peer lifecycle chaincode queryinstalled | \
  grep "$NEW_LABEL" | \
  awk -F 'Package ID: |, Label:' '{print $2}')

# Confirm it worked
echo "✅ Exported CC_PACKAGE_ID:"
echo "$CC_PACKAGE_ID"


