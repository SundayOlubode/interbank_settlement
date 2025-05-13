## Install chaincode
echo "🔄 Installing chaincode $NEW_LABEL on peer $CORE_PEER_LOCALMSPID..."
if peer lifecycle chaincode install "$PACKAGE_DIR/$PACKAGE_NAME"; then
  echo "✅ Chaincode $NEW_LABEL installed on peer $CORE_PEER_LOCALMSPID."
else
  echo "❌ Failed to install chaincode $NEW_LABEL" >&2
  exit 1
fi