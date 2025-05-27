set -x

export FABRIC_CFG_PATH=$(pwd)/../config

CHANNEL_NAME="retailchannel"


# ORDERER_CA=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/cbn.naijachain.org/orderers/orderer.cbn.naijachain.org/msp/tlscacerts/tlsca.cbn.naijachain.org-cert.pem

peer channel create -o orderer.cbn.naijachain.org:7050 -c $CHANNEL_NAME \
  -f ./peer/channel-artifacts/${CHANNEL_NAME}/channel.tx \
  --ordererTLSHostnameOverride orderer.cbn.naijachain.org \
  --outputBlock ./peer/channel-artifacts/${CHANNEL_NAME}/${CHANNEL_NAME}.block \
  --tls $CORE_PEER_TLS_ENABLED --cafile $ORDERER_CA --connTimeout 120s

set +x

# Sign channel transaction
# peer channel signconfigtx -f ./peer/channel-artifacts/retailchannel/channel.tx