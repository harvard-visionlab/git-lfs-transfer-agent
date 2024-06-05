# git-lfs-transfer-agent
custom git-lfs-transfer-agent which uses an s3 bucket for lfs storage using the aws-sdk (no proxy server, just the sdk)

(IN DEVELOPMENT...)

# install

```
git clone https://github.com/harvard-visionlab/git-lfs-transfer-agent.git
cd git-lfs-transfer-agent
go mod init lfs-s3-agent
go get github.com/aws/aws-sdk-go/aws
go get github.com/aws/aws-sdk-go/aws/session
go get github.com/aws/aws-sdk-go/service/s3
go build -o lfs-s3-agent main.go
sudo cp lfs-s3-agent /usr/local/bin/
sudo chmod +x /usr/local/bin/lfs-s3-agent
```

Ensure the LFS_API_KEY and LFS_LAMBDA_FINCTION_URL environment variables are set before performing LFS operations:

`nano ~/.bash_profile`

```
export LFS_CACHE_DIR=/home/jovyan/work/DataLocal/lfs-cache
export LFS_AWS_PROFILE=wasabi
export LFS_AWS_ENDPOINT=https://s3.wasabisys.com
export LFS_AWS_USER=alvarez
export LFS_AWS_REGION=us-east-1
export LFS_S3_BUCKET=visionlab-members
```

```
source ~/.bash_profile
```

optional
```
export LFS_LOGGING=true
export LFS_HASH_LENGTH=16
```

Run tests..
```
go test -v
```

Then set the following Git LFS configuration in your repository: 
```
git config lfs.storage "$LFS_CACHE_DIR"
git config lfs.customtransfer.lfs-agent.path /usr/local/bin/lfs-s3-agent
git config lfs.customtransfer.lfs-agent.args ""
git config lfs.customtransfer.lfs-agent.concurrent true
git config lfs.standalonetransferagent lfs-s3-agent
```

check the settings
```
git lfs env
```

Check remote file metadata
```
aws s3api head-object --bucket visionlab-members --key alvarez/test-b1715442aa.csv --profile wasabi
aws s3api head-object --bucket visionlab-members --key alvarez/b1715442aab3c9c446e4ef884c5a9db61f745faa8a839513c6951f7ea1a1e815 --profile wasabi


```

# TODO
- [ ] rename this repo git-lfs-s3-agent