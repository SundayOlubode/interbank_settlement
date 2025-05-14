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

# Chaincode instantiate command
peer chaincode instantiate \
  -o $ORDERER_ADDRESS \
  --tls $CORE_PEER_TLS_ENABLED \
  --cafile $ORDERER_CA \
  -C $CHANNEL_NAME \
  -n account \
  -v "$CC_VERSION" \
  -c '{"Args":[""]}' \
  --peerAddresses $PEERS_TO_CONNECT_TO \
  --tlsRootCertFiles $TLS_ROOT_CERTS \
#   --collections-config ./contracts/collections-config.json # for private data collection permission config


# Deploy the specified chaincode to the network.

# Usage:
#   peer chaincode instantiate [flags]

# Flags:
#   -C, --channelID string               The channel on which this command should be executed
#       --collections-config string      The fully qualified path to the collection JSON file including the file name
#       --connectionProfile string       Connection profile that provides the necessary connection information for the network. Note: currently only supported for providing peer connection information
#   -c, --ctor string                    Constructor message for the chaincode in JSON format (default "{}")
#   -E, --escc string                    The name of the endorsement system chaincode to be used for this chaincode
#   -h, --help                           help for instantiate
#   -l, --lang string                    Language the chaincode is written in (default "golang")
#   -n, --name string                    Name of the chaincode
#       --peerAddresses stringArray      The addresses of the peers to connect to
#   -P, --policy string                  The endorsement policy associated to this chaincode
#       --tlsRootCertFiles stringArray   If TLS is enabled, the paths to the TLS root cert files of the peers to connect to. The order and number of certs specified should match the --peerAddresses flag
#   -v, --version string                 Version of the chaincode specified in install/instantiate/upgrade commands
#   -V, --vscc string                    The name of the verification system chaincode to be used for this chaincode

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
#       --transient string                    Transient map of arguments in JSON encoding
