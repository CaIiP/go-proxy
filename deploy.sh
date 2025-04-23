#!/bin/bash
set -e

echo "Starting EC2 deployment process..."

# Create application directories
sudo mkdir -p /opt/ec2-client
sudo mkdir -p /opt/enclave-server
sudo mkdir -p /opt/viproxy

# Download application files from S3
echo "Downloading application files from S3..."
aws s3 cp s3://your-bucket/ec2-client.zip /tmp/ec2-client.zip
aws s3 cp s3://your-bucket/enclave-server.zip /tmp/enclave-server.zip
aws s3 cp s3://your-bucket/viproxy.zip /tmp/viproxy.zip

# Extract application files
echo "Extracting application files..."
sudo unzip -o /tmp/ec2-client.zip -d /opt/ec2-client/
sudo unzip -o /tmp/enclave-server.zip -d /opt/enclave-server/
sudo unzip -o /tmp/viproxy.zip -d /opt/viproxy/

# Set permissions
sudo chmod -R 755 /opt/ec2-client
sudo chmod -R 755 /opt/enclave-server
sudo chmod -R 755 /opt/viproxy

# Build EC2 client
echo "Building EC2 client application..."
cd /opt/ec2-client
sudo go build -o client client.go

# Build server for enclave (we'll actually use the Docker build, but this ensures it compiles)
echo "Building enclave server application..."
cd /opt/enclave-server
sudo go build -o server server.go

# Build viproxy
echo "Building viproxy..."
cd /opt/viproxy
sudo go build -o viproxy main.go

# Install Docker if not already installed
if ! command -v docker &> /dev/null; then
    echo "Installing Docker..."
    sudo amazon-linux-extras install docker -y
    sudo systemctl start docker
    sudo systemctl enable docker
    sudo usermod -a -G docker ec2-user
fi

# Install nitro-cli if not already installed
if ! command -v nitro-cli &> /dev/null; then
    echo "Installing nitro-cli..."
    sudo amazon-linux-extras install aws-nitro-enclaves-cli -y
    sudo yum install -y aws-nitro-enclaves-cli-devel
    sudo usermod -a -G ne ec2-user
    sudo systemctl start nitro-enclaves-allocator.service
    sudo systemctl enable nitro-enclaves-allocator.service
fi

# Build enclave Docker image
echo "Building enclave Docker image..."
cd /opt/enclave-server
sudo docker build -t enclave-server:latest .

# Create enclave EIF file
echo "Creating enclave EIF file..."
sudo nitro-cli build-enclave --docker-uri enclave-server:latest --output-file /opt/enclave-server/enclave.eif

# Run enclave
echo "Starting Nitro Enclave..."
ENCLAVE_ID=$(sudo nitro-cli run-enclave --eif-path /opt/enclave-server/enclave.eif --memory 512 --cpu-count 2 --enclave-cid 16 | jq -r '.EnclaveID')
echo "Enclave started with ID: $ENCLAVE_ID"

# Start viproxy
echo "Starting viproxy..."
cd /opt/viproxy
sudo IN_ADDRS="127.0.0.1:8000" OUT_ADDRS="16:8000" nohup ./viproxy > /var/log/viproxy.log 2>&1 &
echo "viproxy started"

# Start EC2 client
echo "Starting EC2 client..."
cd /opt/ec2-client
sudo nohup ./client > /var/log/ec2-client.log 2>&1 &
echo "EC2 client started"

echo "Deployment complete!"