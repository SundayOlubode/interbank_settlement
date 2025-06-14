set -x

docker rm -f \
  peer0.firstbank.naijachain.org \
  peer0.zenithbank.naijachain.org \
  peer0.gtbank.naijachain.org \
  peer0.accessbank.naijachain.org \
  peer0.cbn.naijachain.org \
  orderer.cbn.naijachain.org \
  couchdb0 \
  couchdb1 \
  couchdb2 \
  couchdb3 \
  couchdb4 \

docker volume rm $(docker volume ls -q | grep -i 'naijachain')
docker volume prune
docker network prune


# rm ./contracts/cc-version.txt
# rm -rf ./contracts/cc-packages

rm -rf ./channel-artifacts ./crypto-config

set +x