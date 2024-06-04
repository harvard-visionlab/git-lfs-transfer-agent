package main

import (
    "bufio"
    "bytes"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "testing"
)

func TestAgent(t *testing.T) {
    // Setup the environment variables
    env := []string{
        "LFS_AWS_PROFILE=wasabi",
        "LFS_AWS_ENDPOINT=s3.wasabisys.com",
        "LFS_AWS_USER=alvarez",
        "LFS_AWS_REGION=us-east-1",
        "LFS_S3_BUCKET=visionlab-members",
        "LFS_LOCAL_STORAGE=./lfs_cache",
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
        "oid":   "test-b1715442aa.csv",
        "size":  len(content),
        "path":  testFile,
        "action": map[string]interface{}{
            "href": "s3://visionlab-members/alvarez/test-b1715442aa.csv",
        },
    }

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
        "oid":   "test-b1715442aa.csv",
        "size":  len(content),
        "action": map[string]interface{}{
            "href": "s3://visionlab-members/alvarez/test-b1715442aa.csv",
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
    downloadedFile := filepath.Join("./lfs_cache", "test-b1715442aa.csv")
    downloadedContent, err := os.ReadFile(downloadedFile)
    if err != nil {
        t.Fatalf("Failed to read downloaded file: %v", err)
    }

    if !bytes.Equal(content, downloadedContent) {
        t.Fatalf("Downloaded file content does not match original content")
    }

    // Clean up the test file
    // os.Remove(downloadedFile)
}
