# git-lfs-transfer-agent
custom git-lfts-transfer-agent which adds api-key to git-lfs uploads

# install
```
git clone https://github.com/harvard-visionlab/git-lfs-transfer-agent.git
cd git-lfs-transfer-agent
go mod init git-lfs-transfer-agent
go build -o lfs-transfer-agent main.go
```

Then set the following Git LFS configuration in your repository: 
```
git config lfs.customtransfer.lfs-agent.path ./lfs-transfer-agent
git config lfs.customtransfer.lfs-agent.args ""
git config lfs.customtransfer.lfs-agent.concurrent true
git config lfs.standalonetransferagent lfs-agent
```

Ensure the LFS_API_KEY environment variable is set before performing LFS operations:
```
export LFS_API_KEY=your-secret-api-key
```
