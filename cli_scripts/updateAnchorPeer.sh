# You're submitting a configuration transaction to the channel to:
# Update the channel config with anchor peer(s) for your org.
# Commit this change to the ledger so it’s known network-wide.

# === Required environment variables (set in the container) ===
# CHANNEL_NAME          -> e.g., "retailchannel"
# CORE_PEER_LOCALMSPID -> e.g., "GTBankMSP"
# CORE_PEER_ADDRESS     -> e.g., "peer0.gtbank.naijachain.org:7051"
# CORE_PEER_TLS_ROOTCERT_FILE -> TLS cert path of peer
# CORE_PEER_MSPCONFIGPATH     -> MSP path of Admin@org
# ORDERER_ADDRESS       -> e.g., "orderer.cbn.naijachain.org:7050"
# ORDERER_CA            -> CA cert of orderer node

ORDERER_CA=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/cbn.naijachain.org/orderers/orderer.cbn.naijachain.org/msp/tlscacerts/tlsca.cbn.naijachain.org-cert.pem
ORDERER_ADDRESS=orderer.cbn.naijachain.org:7050
CHANNEL_NAME="retailchannel"
ANCHOR_TX="./peer/channel-artifacts/retailchannel/${CORE_PEER_LOCALMSPID}anchors.tx"

echo ">>> Updating anchor peer for $CORE_PEER_LOCALMSPID on channel $CHANNEL_NAME..."
echo ">>> Using anchor tx: $ANCHOR_TX"

peer channel update \
  -o $ORDERER_ADDRESS \
  -c $CHANNEL_NAME \
  -f $ANCHOR_TX \
  --tls \
  --cafile $ORDERER_CA

# Capture exit code
EXIT_CODE=$?

if [ $EXIT_CODE -ne 0 ]; then
  echo "❌ Failed to update anchor peer for $CORE_PEER_LOCALMSPID on channel $CHANNEL_NAME."
  echo "⚠️  Please check if the anchor peer tx file exists and all environment variables are correctly set."
  exit $EXIT_CODE
else
  echo "✅ Successfully updated anchor peer for $CORE_PEER_LOCALMSPID."
fi
