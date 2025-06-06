COMPOSE_PROJECT_NAME=naijachain         # Sets the Docker Compose project name (prefix for container names)
IMAGE_TAG=2.5.12                   # Specifies the Fabric image tag/version (used in docker-compose.yaml)
SYS_CHANNEL=retail-sys-channel   # Sets the name of the system channel (used during channel creation)
PLATFORM=darwin/arm64             # Ensures compatibility with your system architecture
UNIX_SOCK=/var/run/docker.sock   # Path to Docker's Unix socket, used for Docker CLI and API access

# Go Version
GO_VERSION="1.24.3"