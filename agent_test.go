package main

import (
    "bufio"
    "bytes"
    "encoding/json"
    "fmt"
    "strings"
    "os"
    "os/exec"
    "path/filepath"
    "testing"
)

// getFilenameFromOid runs the git log command and parses the result to get the filename
func getFilenameFromOid(repoPath, oid string) (string, error) {
	// Construct the command
	cmd := exec.Command("git", "log", "-G", fmt.Sprintf("oid sha256:%s", oid), "--name-status")
	cmd.Dir = repoPath // Set the directory to the repository path

	// Run the command and capture the output
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run git log command: %v", err)
	}

	// Parse the output to find the filename
	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		// Find lines starting with 'A' (for added files)
		if strings.HasPrefix(line, "A\t") {
			filename := strings.TrimSpace(strings.TrimPrefix(line, "A"))
			return filename, nil
		}
	}

	return "", fmt.Errorf("filename not found for oid: %s", oid)
}

func TestAgent(t *testing.T) {
    // Setup the environment variables
    env := []string{
        "LFS_LOGGING=true",
        "LFS_AWS_PROFILE=wasabi",
        "LFS_AWS_ENDPOINT=s3.wasabisys.com",
        "LFS_AWS_USER=alvarez",
        "LFS_AWS_REGION=us-east-1",
        "LFS_S3_BUCKET=visionlab-members",
        "LFS_LOCAL_STORAGE=./lfs_cache",
        "LFS_HASH_LENGTH=16",
    }

    // Ensure the local storage directory exists
    err := os.MkdirAll("./lfs_cache", os.ModePerm)
    if err != nil {
        t.Fatalf("Failed to create local storage directory: %v", err)
    }

    // Path to the test file
    testFile := filepath.Join("test_files", "test-b1715442aa.csv")

    // Read the content of the test file
    content, err := os.ReadFile(testFile)
    if err != nil {
        t.Fatalf("Failed to read test file: %v", err)
    }

    // Prepare the upload event
    uploadEvent := map[string]interface{}{
        "event": "upload",
        "oid":   "b1715442aab3c9c446e4ef884c5a9db61f745faa8a839513c6951f7ea1a1e815",
        "size":  len(content),
        "path":  testFile,
        "action": map[string]interface{}{
            "href": "s3://visionlab-members/alvarez/test_files/test-b1715442aa.csv/b1715442aab3c9c446e4ef884c5a9db61f745faa8a839513c6951f7ea1a1e815",
        },
    }
    // {Event:upload Oid:b1715442aab3c9c446e4ef884c5a9db61f745faa8a839513c6951f7ea1a1e815 Size:290 Path:/home/jovyan/work/GitHub/lfs-s3-playground/lfs_cache/objects/b1/71/b1715442aab3c9c446e4ef884c5a9db61f745faa8a839513c6951f7ea1a1e815 Action:{Href: Header:map[]}}
    uploadEventBytes, err := json.Marshal(uploadEvent)
    if err != nil {
        t.Fatalf("Failed to marshal upload event: %v", err)
    }

    // Start the agent process
    cmd := exec.Command("/usr/local/bin/lfs-s3-agent")
    cmd.Env = append(cmd.Env, env...)

    stdin, err := cmd.StdinPipe()
    if err != nil {
        t.Fatalf("Failed to get stdin: %v", err)
    }

    stdout, err := cmd.StdoutPipe()
    if err != nil {
        t.Fatalf("Failed to get stdout: %v", err)
    }

    stderr, err := cmd.StderrPipe()
    if err != nil {
        t.Fatalf("Failed to get stderr: %v", err)
    }

    err = cmd.Start()
    if err != nil {
        t.Fatalf("Failed to start agent: %v", err)
    }

    // Write the upload event to the agent's stdin
    stdin.Write(uploadEventBytes)
    stdin.Write([]byte("\n"))
    stdin.Close()

    // Read the response from the agent's stdout
    scanner := bufio.NewScanner(stdout)
    if scanner.Scan() {
        response := scanner.Text()
        fmt.Printf("Upload Response: %s\n", response)
    }

    // Read any error output
    errOutput := new(bytes.Buffer)
    errOutput.ReadFrom(stderr)
    if errOutput.Len() > 0 {
        t.Fatalf("Agent error output: %s", errOutput.String())
    }

    // Wait for the agent to finish
    err = cmd.Wait()
    if err != nil {
        t.Fatalf("Agent process error: %v", err)
    }

    // Prepare the download event
    downloadEvent := map[string]interface{}{
        "event": "download",
        "oid":   "b1715442aab3c9c446e4ef884c5a9db61f745faa8a839513c6951f7ea1a1e815",
        "size":  len(content),
        "action": map[string]interface{}{
            "href": "s3://visionlab-members/alvarez/test_files/test-b1715442aa.csv/b1715442aab3c9c446e4ef884c5a9db61f745faa8a839513c6951f7ea1a1e815",
        },
    }

    downloadEventBytes, err := json.Marshal(downloadEvent)
    if err != nil {
        t.Fatalf("Failed to marshal download event: %v", err)
    }

    // Start the agent process for download
    cmd = exec.Command("/usr/local/bin/lfs-s3-agent")
    cmd.Env = append(cmd.Env, env...)

    stdin, err = cmd.StdinPipe()
    if err != nil {
        t.Fatalf("Failed to get stdin: %v", err)
    }

    stdout, err = cmd.StdoutPipe()
    if err != nil {
        t.Fatalf("Failed to get stdout: %v", err)
    }

    stderr, err = cmd.StderrPipe()
    if err != nil {
        t.Fatalf("Failed to get stderr: %v", err)
    }

    err = cmd.Start()
    if err != nil {
        t.Fatalf("Failed to start agent: %v", err)
    }

    // Write the download event to the agent's stdin
    stdin.Write(downloadEventBytes)
    stdin.Write([]byte("\n"))
    stdin.Close()

    // Read the response from the agent's stdout
    scanner = bufio.NewScanner(stdout)
    if scanner.Scan() {
        response := scanner.Text()
        fmt.Printf("Download Response: %s\n", response)
    }

    // Read any error output
    errOutput = new(bytes.Buffer)
    errOutput.ReadFrom(stderr)
    if errOutput.Len() > 0 {
        t.Fatalf("Agent error output: %s", errOutput.String())
    }

    // Wait for the agent to finish
    err = cmd.Wait()
    if err != nil {
        t.Fatalf("Agent process error: %v", err)
    }

    // Verify the downloaded file
    downloadedFile := filepath.Join("./lfs_cache", "b1715442aab3c9c4", "test-b1715442aa.csv")
    downloadedContent, err := os.ReadFile(downloadedFile)
    if err != nil {
        t.Fatalf("Failed to read downloaded file: %v", err)
    }

    if !bytes.Equal(content, downloadedContent) {
        t.Fatalf("Downloaded file content does not match original content")
    }
    
    // filename
    filename, err := getFilenameFromOid("/home/jovyan/work/GitHub/lfs-s3-playground",
                                        "b1715442aab3c9c446e4ef884c5a9db61f745faa8a839513c6951f7ea1a1e815")
    fmt.Printf("Recovered filename: %s\n", filename)
    
    // Clean up the test file
    // os.Remove(downloadedFile)
}
