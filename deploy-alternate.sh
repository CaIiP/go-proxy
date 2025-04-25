#!/bin/bash
set -e
set -x  # Print each command as it's executed

echo "Starting EC2 deployment process..."

# Create application directories
sudo mkdir -p /opt/ec2-client
sudo mkdir -p /opt/enclave-server
sudo mkdir -p /opt/viproxy

# Create log directory with proper permissions
sudo mkdir -p /var/log
sudo chmod 777 /var/log

# Download pre-built applications from S3
echo "Downloading pre-built applications from S3..."
aws s3 cp s3://your-bucket/ec2-client /opt/ec2-client/client
aws s3 cp s3://your-bucket/enclave-server /opt/enclave-server/server
aws s3 cp s3://your-bucket/viproxy /opt/viproxy/viproxy

# Set executable permissions
sudo chmod +x /opt/ec2-client/client
sudo chmod +x /opt/enclave-server/server
sudo chmod +x /opt/viproxy/viproxy

# Create Dockerfile for enclave
echo "Creating Dockerfile for enclave..."
cat > /opt/enclave-server/Dockerfile << 'EOF'
FROM amazonlinux:2

# Create app directory
WORKDIR /app

# Copy pre-built binary
COPY server /app/

# Expose the port
EXPOSE 8000

# Set the entry point
ENTRYPOINT ["/app/server"]
EOF

# Install Docker if not already installed
if ! command -v docker &> /dev/null; then
    echo "Installing Docker..."
    sudo yum install -y docker
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
ENCLAVE_ID=$(sudo nitro-cli run-enclave --eif-path /opt/enclave-server/enclave.eif --memory 512 --cpu-count 2 --enclave-cid 16 | grep -o '"EnclaveID": "[^"]*"' | cut -d'"' -f4)
echo "Enclave started with ID: $ENCLAVE_ID"

# Start viproxy
echo "Starting viproxy..."
cd /opt/viproxy
sudo bash -c 'IN_ADDRS="127.0.0.1:8000" OUT_ADDRS="16:8000" nohup ./viproxy > /var/log/viproxy.log 2>&1 &'
echo "viproxy started"

# Start EC2 client
echo "Starting EC2 client..."
cd /opt/ec2-client
sudo bash -c 'nohup ./client > /var/log/ec2-client.log 2>&1 &'
echo "EC2 client started"

echo "Deployment complete!"
echo "Check logs at:"
echo "  - /var/log/viproxy.log"
echo "  - /var/log/ec2-client.log"

# Print basic status information
echo "Checking if enclave is running:"
sudo nitro-cli describe-enclaves

echo "Checking if viproxy is running:"
ps aux | grep viproxy | grep -v grep

echo "Checking if EC2 client is running:"
ps aux | grep client | grep -v grep