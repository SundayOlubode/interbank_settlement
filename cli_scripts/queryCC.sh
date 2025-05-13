## Query Installed Chaincodes
echo "🔍 Querying installed chaincodes on peer $CORE_PEER_LOCALMSPID..."
QUERY_OUTPUT=$(peer lifecycle chaincode queryinstalled 2>&1)
if [ $? -ne 0 ]; then
  echo "❌ Failed to query installed chaincodes." >&2
  echo "$QUERY_OUTPUT"
  exit 1
else
  echo "✅ Installed chaincodes:"
  echo "$QUERY_OUTPUT"
fi