# Building and Uploading Applications

## 1. Building the Applications

Build each application on your local machine or any environment with Go installed:

### EC2 Client
```bash
cd ec2-client
go build -o ec2-client ./src
```

### Enclave Server
```bash
cd enclave-server
GOOS=linux GOARCH=amd64 go build -o enclave-server ./src
```

### VIProxy
```bash
cd viproxy
go build -o viproxy ./src
```

## 2. Uploading to S3

Upload the built binaries to your S3 bucket:

```bash
aws s3 cp ec2-client/ec2-client s3://your-bucket/ec2-client
aws s3 cp enclave-server/enclave-server s3://your-bucket/enclave-server
aws s3 cp viproxy/viproxy s3://your-bucket/viproxy
```

Make sure to replace `your-bucket` with your actual S3 bucket name.

## 3. Deploying to EC2

1. Connect to your EC2 instance
2. Create a file called `deploy.sh` with the content of the simplified deployment script
3. Make it executable: `chmod +x deploy.sh`
4. Execute it: `./deploy.sh`

## 4. Verification

After deployment completes, check the logs:

```bash
sudo tail -f /var/log/ec2-client.log
sudo tail -f /var/log/viproxy.log
```

Check if the enclave is running:

```bash
sudo nitro-cli describe-enclaves
```