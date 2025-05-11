ORDERER_CA=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/cbn.naijachain.org/orderers/orderer.cbn.naijachain.org/msp/tlscacerts/tlsca.cbn.naijachain.org-cert.pem


peer channel create -o orderer.cbn.naijachain.org:7050 -c retailchannel -f ./channel-artifacts/channel.tx --outputBlock ./channel.block --tls $CORE_PEER_TLS_ENABLED --cafile ${ORDERER_CA} --connTimeout 120s >&log.txt
