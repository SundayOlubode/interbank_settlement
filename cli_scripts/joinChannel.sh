set -x

CHANNEL_NAME="retailchannel"

peer channel join -b ./peer/channel-artifacts/${CHANNEL_NAME}/${CHANNEL_NAME}.block

set +x