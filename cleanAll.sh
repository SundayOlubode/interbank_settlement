docker rm -f \
  peer0.firstbank.naijachain.org \
  peer0.zenithbank.naijachain.org \
  peer0.gtbank.naijachain.org \
  peer0.accessbank.naijachain.org \
  orderer.cbn.naijachain.org

docker volume rm $(docker volume ls -q | grep -i 'naijachain')

rm ./contracts/cc-version.txt
rm -rf ./contracts/cc-packages



# ./peer/scripts/createChannel.sh
# ./peer/scripts/joinChannel.sh
# ./peer/scripts/updateAnchorPeer.sh
# ./peer/scripts/packageCC.sh
# ./peer/scripts/installCC.sh
# ./peer/scripts/approve-single-org.sh
# ./peer/scripts/check-commit-readiness.sh
# ./peer/scripts/commit-chaincode.sh


