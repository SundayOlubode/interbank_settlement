set -e

VERSION_FILE="/opt/gopath/src/github.com/hyperledger/fabric/peer/contracts/cc-version.txt"
ORDERER_CA="/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/cbn.naijachain.org/orderers/orderer.cbn.naijachain.org/msp/tlscacerts/tlsca.cbn.naijachain.org-cert.pem"
ORDERER_ADDRESS="orderer.cbn.naijachain.org:7050"
CHANNEL_NAME="retailchannel"
PEERS_TO_CONNECT_TO=""
TLS_ROOT_CERT_BASE_DIR="/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations"
TLS_ROOT_CERTS=""

# Assume CORE_PEER_ADDRESS is set
: "${CORE_PEER_ADDRESS:?Please set CORE_PEER_ADDRESS before running this script.}"

PEERS=(
  "peer0.accessbank.naijachain.org:7051"
  "peer0.gtbank.naijachain.org:8051"
  "peer0.zenithbank.naijachain.org:9051"
  "peer0.firstbank.naijachain.org:10051"
)

# Loop through peers
for PEER in "${PEERS[@]}"; do
  if [ "$PEER" != "$CORE_PEER_ADDRESS" ]; then
    PEERS_TO_CONNECT_TO+="$PEER "
    
    ORG=$(echo "$PEER" | cut -d'.' -f2)  # extract org like 'gtbank'
    PEER_NAME=$(echo "$PEER" | cut -d':' -f1)  # extract peer name like 'peer0.gtbank...'
    
    TLS_ROOT_CERTS+=" $TLS_ROOT_CERT_BASE_DIR/$ORG.naijachain.org/peers/$PEER_NAME/tls/ca.crt"
  fi
done

# Trim
PEERS_TO_CONNECT_TO=$(echo "$PEERS_TO_CONNECT_TO" | xargs)
TLS_ROOT_CERTS=$(echo "$TLS_ROOT_CERTS" | xargs)

# Output for debugging
echo "🧩 Other peer addresses: $PEERS_TO_CONNECT_TO"
echo "📜 TLS certs: $TLS_ROOT_CERTS"

# Read chaincode version
CC_VERSION=$(sed -n '1p' "$VERSION_FILE")

# peer lifecycle chaincode approveformyorg --connTimeout 120s -o $ORDERER_HOST:$ORDERER_PORT --ordererTLSHostnameOverride $ORDERER_HOST --channelID $CHANNEL_NAME --name $CC_LABEL --version $CC_VERSION --signature-policy $ENDORSEMENT_POLICY --init-required --package-id $CC_PKG_ID --sequence $CC_SEQUENCE_NUM --tls true --cafile $ORDERER_CA >&log.txt


# Chaincode approveformyorg command
peer lifecycle chaincode approveformyorg\
  --connTimeout 120s \
  -C $CHANNEL_NAME \
  -n account \
  --package-id $CC_PACKAGE_ID \
  --peerAddresses $PEERS_TO_CONNECT_TO \
  --sequence 1 \ # Continue from here
  -o $ORDERER_ADDRESS \
  --tls $CORE_PEER_TLS_ENABLED \
  --cafile $ORDERER_CA \
  -v "$CC_VERSION" \
  -c '{"Args":[""]}' \
  --tlsRootCertFiles $TLS_ROOT_CERTS \


# Approve the chaincode definition for my organization.

# Usage:
#   peer lifecycle chaincode approveformyorg [flags]

# Flags:
#       --channel-config-policy string   The endorsement policy associated to this chaincode specified as a channel config policy reference
#   -C, --channelID string               The channel on which this command should be executed
#       --collections-config string      The fully qualified path to the collection JSON file including the file name
#       --connectionProfile string       The fully qualified path to the connection profile that provides the necessary connection information for the network. Note: currently only supported for providing peer connection information
#   -E, --endorsement-plugin string      The name of the endorsement plugin to be used for this chaincode
#   -h, --help                           help for approveformyorg
#       --init-required                  Whether the chaincode requires invoking 'init'
#   -n, --name string                    Name of the chaincode
#       --package-id string              The identifier of the chaincode install package
#       --peerAddresses stringArray      The addresses of the peers to connect to
#       --sequence int                   The sequence number of the chaincode definition for the channel
#       --signature-policy string        The endorsement policy associated to this chaincode specified as a signature policy
#       --tlsRootCertFiles stringArray   If TLS is enabled, the paths to the TLS root cert files of the peers to connect to. The order and number of certs specified should match the --peerAddresses flag
#   -V, --validation-plugin string       The name of the validation plugin to be used for this chaincode
#   -v, --version string                 Version of the chaincode
#       --waitForEvent                   Whether to wait for the event from each peer's deliver filtered service signifying that the transaction has been committed successfully (default true)
#       --waitForEventTimeout duration   Time to wait for the event from each peer's deliver filtered service signifying that the 'invoke' transaction has been committed successfully (default 30s)

# Global Flags:
#       --cafile string                       Path to file containing PEM-encoded trusted certificate(s) for the ordering endpoint
#       --certfile string                     Path to file containing PEM-encoded X509 public key to use for mutual TLS communication with the orderer endpoint
#       --clientauth                          Use mutual TLS when communicating with the orderer endpoint
#       --connTimeout duration                Timeout for client to connect (default 3s)
#       --keyfile string                      Path to file containing PEM-encoded private key to use for mutual TLS communication with the orderer endpoint
#   -o, --orderer string                      Ordering service endpoint
#       --ordererTLSHostnameOverride string   The hostname override to use when validating the TLS connection to the orderer
#       --tls                                 Use TLS when communicating with the orderer endpoint
#       --tlsHandshakeTimeShift duration      The amount of time to shift backwards for certificate expiration checks during TLS handshakes with the orderer endpoint
