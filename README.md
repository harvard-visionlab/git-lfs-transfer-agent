# git-lfs-transfer-agent
custom git-lfts-transfer-agent which adds api-key to git-lfs uploads

# install

```
git clone https://github.com/harvard-visionlab/git-lfs-transfer-agent.git
cd git-lfs-transfer-agent
go mod init git-lfs-transfer-agent
go build -o lfs-transfer-agent main.go
sudo cp lfs-transfer-agent /usr/local/bin/
sudo chmod +x /usr/local/bin/lfs-transfer-agent
```

take2
```
go mod init lfs-s3-agent
go get github.com/aws/aws-sdk-go/aws
go get github.com/aws/aws-sdk-go/aws/session
go get github.com/aws/aws-sdk-go/service/s3
go build -o lfs-s3-agent main.go
sudo cp lfs-s3-agent /usr/local/bin/
sudo chmod +x /usr/local/bin/lfs-s3-agent
```

Then set the following Git LFS configuration in your repository: 
```
git config lfs.customtransfer.lfs-agent.path /usr/local/bin/lfs-transfer-agent
git config lfs.customtransfer.lfs-agent.args ""
git config lfs.customtransfer.lfs-agent.concurrent true
git config lfs.standalonetransferagent lfs-agent
```

Run tests..
```
go test -v
```

Check remote file metadata
```
aws s3api head-object --bucket visionlab-members --key alvarez/test-b1715442aa.csv --profile wasabi
```

Ensure the LFS_API_KEY and LFS_LAMBDA_FINCTION_URL environment variables are set before performing LFS operations:
```
export LFS_API_KEY=your-secret-api-key
export LFS_LAMBDA_FINCTION_URL=url-to-your-lfs-s3-lambda function
```

```
export LFS_AWS_PROFILE=wasabi
export LFS_AWS_ENDPOINT=s3.wasabisys.com
export LFS_AWS_USER=alvarez
export LFS_AWS_REGION=us-east-1
export LFS_LOCAL_STORAGE=/path/to/local/storage
```