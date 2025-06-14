set -e
./cleanAll.sh

sleep 2

./generate.sh
sleep 2

./create-channel.sh
sleep 2

./deploy-cc.sh
sleep 2

./invoke-cc.sh