ORDERER_CA="./crypto-config/ordererOrganizations/cbn.naijachain.org/orderers/orderer.cbn.naijachain.org/msp/tlscacerts/tlsca.cbn.naijachain.org-cert.pem"
CORE_PEER_TLS_ROOTCERT_FILE=./crypto-config/peerOrganizations/accessbank.naijachain.org/peers/peer0.accessbank.naijachain.org/tls/ca.crt


# openssl x509 -noout -text -in $ORDERER_CA | grep -A1 "Subject Alternative Name"
openssl x509 -noout -text -in $CORE_PEER_TLS_ROOTCERT_FILE | grep -A1 "Subject Alternative Name"
