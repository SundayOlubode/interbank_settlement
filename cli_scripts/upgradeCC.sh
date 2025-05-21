## Upgrade chaincode

BASE_PATH="/opt/gopath/src/github.com/hyperledger/fabric/peer"

set -e # Exit on error

# Track Chaincode Version
CHAINCODE_NAME="account"
CHAINCODE_LANG="golang"
CHAINCODE_PATH="$BASE_PATH/contracts"
CHAINCODE_LABEL_BASE="account"

PACKAGE_DIR="$CHAINCODE_PATH/cc-packages"
VERSION_FILE="$CHAINCODE_PATH/cc-version.txt"

mkdir -p "$PACKAGE_DIR"

# === Determine new version and sequence ===
if [ -f "$VERSION_FILE" ]; then
  CURRENT_VERSION=$(head -n 1 "$VERSION_FILE")
  IFS='.' read -r MAJOR MINOR <<<"$CURRENT_VERSION"
  NEW_MINOR=$((MINOR + 1))
  NEW_VERSION="${MAJOR}.${NEW_MINOR}"

  CURRENT_SEQUENCE=$(grep SEQUENCE_NUM "$VERSION_FILE" | cut -d'=' -f2)
  if [ -n "$CURRENT_SEQUENCE" ]; then
    NEW_SEQUENCE_NUM=$((CURRENT_SEQUENCE + 1))
  else
    NEW_SEQUENCE_NUM=1
  fi
else
  CURRENT_VERSION="1.0"
  NEW_VERSION=$CURRENT_VERSION
  NEW_SEQUENCE_NUM=1
fi

NEW_LABEL="${CHAINCODE_LABEL_BASE}_${NEW_VERSION}"
PACKAGE_NAME="${NEW_LABEL}.tar.gz"
PACKAGE_PATH="${PACKAGE_DIR}/${PACKAGE_NAME}"

# === Save new version metadata ===
{
  echo "$NEW_VERSION"
  echo "LABEL=$NEW_LABEL"
  echo "PACKAGE_NAME=$PACKAGE_NAME"
  echo "PACKAGE_PATH=$PACKAGE_PATH"
  echo "SEQUENCE_NUM=$NEW_SEQUENCE_NUM"
} > "$VERSION_FILE"

echo "✅ Chaincode upgrade version details saved to $VERSION_FILE"
