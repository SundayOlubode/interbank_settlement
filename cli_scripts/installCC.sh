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


# Install a chaincode on a peer. This installs a chaincode deployment spec package (if provided) or packages the specified chaincode before subsequently installing it.

# Usage:
#   peer chaincode install [flags]

# Flags:
#       --connectionProfile string       Connection profile that provides the necessary connection information for the network. Note: currently only supported for providing peer connection information
#   -c, --ctor string                    Constructor message for the chaincode in JSON format (default "{}")
#   -h, --help                           help for install
#   -l, --lang string                    Language the chaincode is written in (default "golang")
#   -n, --name string                    Name of the chaincode
#   -p, --path string                    Path to chaincode
#       --peerAddresses stringArray      The addresses of the peers to connect to
#       --tlsRootCertFiles stringArray   If TLS is enabled, the paths to the TLS root cert files of the peers to connect to. The order and number of certs specified should match the --peerAddresses flag
#   -v, --version string                 Version of the chaincode specified in install/instantiate/upgrade commands

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