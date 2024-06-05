# git-lfs-transfer-agent
custom git-lfs-transfer-agent which uses an s3 bucket for lfs storage using the aws-sdk (no proxy server, just the sdk)

(IN DEVELOPMENT...)

# install git lfs
https://github.com/git-lfs/git-lfs/releases

Determine which version to download based on your system:
```
uname -m
```

x86_64 indicates an amd64 architecture.

```
cd ~/tmp
wget -c https://github.com/git-lfs/git-lfs/releases/download/v3.5.1/git-lfs-linux-amd64-v3.5.1.tar.gz
tar -xvzf git-lfs-linux-amd64-v3.5.1.tar.gz
cd git-lfs-3.5.1
PREFIX=$HOME/local ./install.sh
```

check installation
```
git lfs version
```

# make sure go is installed on your system
```
go version
```

If not go [here](https://go.dev/dl/) to download the latest binary:
```
cd ~/tmp
wget -c https://go.dev/dl/go1.22.4.linux-amd64.tar.gz
tar -xzf go1.22.4.linux-amd64.tar.gz

```

append to bashrc `nano ~/.bashrc`
```
export PATH=$HOME/go/bin:$PATH
```

```
source ~/.bashrc
go version
```

# install git-lfs-transfer-agent

```
git clone https://github.com/harvard-visionlab/git-lfs-transfer-agent.git
cd git-lfs-transfer-agent
go mod init lfs-s3-agent
go get github.com/aws/aws-sdk-go/aws
go get github.com/aws/aws-sdk-go/aws/session
go get github.com/aws/aws-sdk-go/service/s3
go build -o lfs-s3-agent main.go
```

Copy to /user/local/bin
```
sudo cp lfs-s3-agent /usr/local/bin/
sudo chmod +x /usr/local/bin/lfs-s3-agent
```

or to `$HOME/local/bin`
```
cp lfs-s3-agent $HOME/local/bin
chmod +x $HOME/local/bin/lfs-s3-agent
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

or globally
```
git config --global lfs.storage "$LFS_CACHE_DIR"
git config --global lfs.customtransfer.lfs-s3-agent.path /usr/local/bin/lfs-s3-agent
git config --global lfs.customtransfer.lfs-s3-agent.path $HOME/local/bin/lfs-s3-agent
git config --global lfs.customtransfer.lfs-s3-agent.args ""
git config --global lfs.customtransfer.lfs-s3-agent.concurrent true
git config --global lfs.standalonetransferagent lfs-s3-agent
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