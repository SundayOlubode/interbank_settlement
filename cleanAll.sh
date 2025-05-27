set -x

docker rm -f \
  peer0.firstbank.naijachain.org \
  peer0.zenithbank.naijachain.org \
  peer0.gtbank.naijachain.org \
  peer0.accessbank.naijachain.org \
  orderer.cbn.naijachain.org

docker volume rm $(docker volume ls -q | grep -i 'naijachain')

rm ./contracts/cc-version.txt
rm -rf ./contracts/cc-packages

set +x

# ./peer/scripts/createChannel.sh
# ./peer/scripts/joinChannel.sh
# ./peer/scripts/updateAnchorPeer.sh
# ./peer/scripts/packageCC.sh
# ./peer/scripts/installCC.sh
# ./peer/scripts/approve-single-org.sh
# ./peer/scripts/check-commit-readiness.sh
# ./peer/scripts/commit-chaincode.sh


# ./peer/scripts/createChannel.sh && ./peer/scripts/joinChannel.sh && ./peer/scripts/updateAnchorPeer.sh


# peer lifecycle chaincode approveformyorg \
#   -o orderer.cbn.naijachain.org:7050 \
#   --ordererTLSHostnameOverride orderer.cbn.naijachain.org \
#   --channelID retailchannel \
#   --name account --version 1.0 --sequence 1 \
#   --package-id account_1.0:… \
#   --tls --cafile $ORDERER_CA \
#   --peerAddresses peer0.accessbank.naijachain.org:7051 \
#   --tlsRootCertFiles /etc/hyperledger/fabric/tls/ca.crt \
#   --peerAddresses peer0.gtbank.naijachain.org:8051 \
#   --tlsRootCertFiles /etc/hyperledger/fabric/crypto/peerOrganizations/gtbank.naijachain.org/peers/peer0.gtbank.naijachain.org/tls/ca.crt \
#   --peerAddresses peer0.zenithbank.naijachain.org:9051 \
#   --tlsRootCertFiles /etc/hyperledger/fabric/crypto/peerOrganizations/zenithbank.naijachain.org/peers/peer0.zenithbank.naijachain.org/tls/ca.crt
