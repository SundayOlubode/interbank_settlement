#!/bin/bash

#Import environment variables
source ./env-vars.sh

CHANNEL_NAME=$1
NEW_BATCH_SIZE=${2:-50}

setGlobalForPeer0AccessBank

set -x
peer channel fetch config config_block.pb -o localhost:7050 -c $CHANNEL_NAME --tls --cafile $ORDERER_CA

configtxlator proto_decode --input config_block.pb --type common.Block --output config_block.json
jq .data.data[0].payload.data.config config_block.json > config.json


MAXBATCHSIZEPATH=".channel_group.groups.Orderer.values.BatchSize.value.max_message_count"

echo "current batch size is $(jq "$MAXBATCHSIZEPATH" config.json)"

jq "$MAXBATCHSIZEPATH = $NEW_BATCH_SIZE" config.json > modified_config.json


configtxlator proto_encode --input config.json --type common.Config --output config.pb
configtxlator proto_encode --input modified_config.json --type common.Config --output modified_config.pb

configtxlator compute_update --channel_id $CHANNEL_NAME --original config.pb --updated modified_config.pb --output final_update.pb

configtxlator proto_decode --input final_update.pb --type common.ConfigUpdate --output final_update.json

echo "{\"payload\":{\"header\":{\"channel_header\":{\"channel_id\":\"$CHANNEL_NAME\", \"type\":2}},\"data\":{\"config_update\":"$(cat final_update.json)"}}}" | jq . >  header_in_envolope.json

configtxlator proto_encode --input header_in_envolope.json --type common.Envelope --output final_update_in_envelope.pb

peer channel signconfigtx -f final_update_in_envelope.pb

CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/crypto-config/ordererOrganizations/cbn.naijachain.org/orderers/orderer.cbn.naijachain.org/tls/ca.crt
CORE_PEER_MSPCONFIGPATH=${PWD}/crypto-config/ordererOrganizations/cbn.naijachain.org/users/Admin@cbn.naijachain.org/msp
CORE_PEER_ADDRESS=localhost:7050
CORE_PEER_LOCALMSPID=CentralBankMSP

peer channel update -f final_update_in_envelope.pb -c $CHANNEL_NAME -o localhost:7050 --tls --cafile $ORDERER_CA