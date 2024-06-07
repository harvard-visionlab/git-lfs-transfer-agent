package main

import (
    "crypto/sha256"
    "encoding/hex"
    "bufio"
    "encoding/json"
    "fmt"
    "strings"
    "io"
    "log"
    "os"
    "path/filepath"
    "strconv"
    
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/credentials"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
)

const defaultLfsStorage = ".git/lfs/objects"

type InitEvent struct {
    Event               string `json:"event"`
    Operation           string `json:"operation"`
    Remote              string `json:"remote"`
    Concurrent          bool   `json:"concurrent"`
    ConcurrentTransfers int    `json:"concurrenttransfers"`
}

type TransferEvent struct {
    Event  string `json:"event"`
    Oid    string `json:"oid"`
    Size   int64  `json:"size"`
    Path   string `json:"path"`
    Action struct {
        Href   string            `json:"href"`
        Header map[string]string `json:"header"`
    } `json:"action"`
}

type CompleteEvent struct {
    Event string `json:"event"`
    Oid   string `json:"oid"`
    Path  string `json:"path,omitempty"`
    Error struct {
        Code    int    `json:"code"`
        Message string `json:"message"`
    } `json:"error,omitempty"`
}

func handleInit(event InitEvent) {
    response := map[string]string{}
    sendResponse(response)
}

func handleUpload(event TransferEvent, svc *s3.S3) {
    loggingEnabled := os.Getenv("LFS_LOGGING") == "true"
    
    // Set log output to stderr
    log.SetOutput(os.Stderr)

    // Direct print statement for debugging
    fmt.Fprintf(os.Stderr, "==> Entered handleUpload function, loggingEnabled: %t\n", loggingEnabled)
    
    if loggingEnabled {
        fmt.Fprintf(os.Stderr, "==> Received upload event: %+v\n", event)
    }

    // Check if the object already exists
    key := fmt.Sprintf("%s/git_lfs/objects/%s", os.Getenv("LFS_AWS_USER"), event.Oid)
    headObjInput := &s3.HeadObjectInput{
        Bucket: aws.String(os.Getenv("LFS_S3_BUCKET")),
        Key:    aws.String(key),
    }

    headObjOutput, err := svc.HeadObject(headObjInput)
    if err == nil {
        // Get the SHA-256 hash from metadata
        remoteSHA256 := headObjOutput.Metadata["Sha256"]

        // Compare hashes
        if remoteSHA256 != nil && *remoteSHA256 == event.Oid {
            if loggingEnabled {
                fmt.Fprintf(os.Stderr, "==> oid %s already exists at key %s with matching SHA-256, skipping upload\n", event.Oid, key)
            }
            // The object exists and the hash matches, no need to upload
            response := CompleteEvent{Event: "complete", Oid: event.Oid}
            sendResponse(response)
            return
        }
    }

    // Proceed to upload the object
    file, err := os.Open(event.Path)
    if err != nil {
        handleError(event, fmt.Errorf("failed to open file %q: %v", event.Path, err))
        return
    }
    defer file.Close()

    /// Log when the upload starts
    fmt.Fprintf(os.Stderr, "Uploading file %s to s3://%s/%s\t", event.Path, os.Getenv("LFS_S3_BUCKET"), key)
    
    //  Get ACL, default to private 
    acl := os.Getenv("LFS_ACL")
    if acl == "" {
        acl = "private"
    }
    
    // Upload
    _, err = svc.PutObject(&s3.PutObjectInput{
        Bucket: aws.String(os.Getenv("LFS_S3_BUCKET")),
        Key:    aws.String(key),
        Body:   file,
        Metadata: map[string]*string{
            "Sha256": aws.String(event.Oid),
        },
        ACL: aws.String(acl),
    })
    if err != nil {
        handleError(event, fmt.Errorf("failed to upload data to %s/%s: %v", os.Getenv("LFS_S3_BUCKET"), key, err))
        return
    }
    
    // Log when the upload is successful
    fmt.Fprintf(os.Stderr, "Successfully uploaded file %s to s3://%s/%s\n", event.Path, os.Getenv("LFS_S3_BUCKET"), key)

    response := CompleteEvent{Event: "complete", Oid: event.Oid}
    sendResponse(response)
}

// Helper function to handle errors and send a response
func handleError(event TransferEvent, err error) {
    response := CompleteEvent{
        Event: "complete",
        Oid:   event.Oid,
        Error: struct {
            Code    int    `json:"code"`
            Message string `json:"message"`
        }{
            Code:    1,
            Message: err.Error(),
        },
    }
    sendResponse(response)
}

func computeSHA256(filePath string) (string, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return "", err
    }
    defer file.Close()

    hasher := sha256.New()
    if _, err := io.Copy(hasher, file); err != nil {
        return "", err
    }

    return hex.EncodeToString(hasher.Sum(nil)), nil
}

// Helper function to get the hash length from the environment variable or default to 16
func getHashLength() int {
    hashLengthStr := os.Getenv("LFS_HASH_LENGTH")
    if hashLengthStr == "" {
        return 16
    }

    hashLength, err := strconv.Atoi(hashLengthStr)
    if err != nil {
        return 16
    }
    return hashLength
}

// Helper function to update the metadata of an S3 object
func updateS3ObjectMetadata(svc *s3.S3, bucket, key, sha256Hash string) error {
    _, err := svc.CopyObject(&s3.CopyObjectInput{
        Bucket:     aws.String(bucket),
        CopySource: aws.String(fmt.Sprintf("%s/%s", bucket, key)),
        Key:        aws.String(key),
        Metadata: map[string]*string{
            "Sha256": aws.String(sha256Hash),
        },
        MetadataDirective: aws.String("REPLACE"),
    })
    if err != nil {
        return fmt.Errorf("failed to update metadata for %s/%s: %v", bucket, key, err)
    }
    return nil
}

// Helper function to download a file from S3 to a local path
func downloadFileFromS3(svc *s3.S3, bucket, key, localPath string) error {
    output, err := svc.GetObject(&s3.GetObjectInput{
        Bucket: aws.String(bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        return fmt.Errorf("failed to download data from %s/%s: %v", bucket, key, err)
    }
    defer output.Body.Close()

    // Ensure the directory exists
    if err = os.MkdirAll(filepath.Dir(localPath), os.ModePerm); err != nil {
        return fmt.Errorf("failed to create directory for file %q: %v", localPath, err)
    }

    // Create the local file
    file, err := os.Create(localPath)
    if err != nil {
        return fmt.Errorf("failed to create file %q: %v", localPath, err)
    }
    defer file.Close()

    // Copy the S3 object to the local file
    if _, err = io.Copy(file, output.Body); err != nil {
        return fmt.Errorf("failed to write data to file %q: %v", localPath, err)
    }

    return nil
}

// Helper function to parse the event.action.href
func parseHref(href string) (string, string, string, error) {
    // trim bucket prefix
    bucketPrefix := fmt.Sprintf("s3://%s", os.Getenv("LFS_S3_BUCKET"))
    key := strings.TrimPrefix(href, bucketPrefix)

    // Find the last '/' to split the oid and filename
    idx := strings.LastIndex(key, "/")
    if idx == -1 {
        return "", "", "", fmt.Errorf("invalid S3 key format")
    }

    oid := key[idx+1:]
    filename := filepath.Base(key[:idx])

    return key, filename, oid, nil
}

func handleDownload(event TransferEvent, svc *s3.S3) {
    loggingEnabled := os.Getenv("LFS_LOGGING") == "true"

    if loggingEnabled {
        fmt.Fprintf(os.Stderr, "==> Received download event: %+v\n", event)
    }
    
    // Temporarily store the file in the LFS_CACHE_DIR/tmp folder
    localStorage := os.Getenv("LFS_CACHE_DIR")
    if localStorage == "" {
        localStorage = defaultLfsStorage
    }
    tempFile := filepath.Join(localStorage, "tmp", event.Oid)
    
    // Download the object from the bucket
    bucket := os.Getenv("LFS_S3_BUCKET")
    key := fmt.Sprintf("%s/git_lfs/objects/%s", os.Getenv("LFS_AWS_USER"), event.Oid)
    
    if loggingEnabled {
        fmt.Fprintf(os.Stderr, "==> Downloading file from s3://%s/%s to %s\n", bucket, key, tempFile)
    }
    err := downloadFileFromS3(svc, bucket, key, tempFile)
    if err != nil {
        handleError(event, err)
        return
    }

    response := CompleteEvent{Event: "complete", Oid: event.Oid, Path: tempFile}
    sendResponse(response)
}

func sendResponse(response interface{}) {
    if completeEvent, ok := response.(CompleteEvent); ok {
        if completeEvent.Error.Code == 0 && completeEvent.Error.Message == "" {
            // Marshal without the Error field
            jsonResponse, err := json.Marshal(struct {
                Event string `json:"event"`
                Oid   string `json:"oid"`
                Path  string `json:"path,omitempty"`
            }{
                Event: completeEvent.Event,
                Oid:   completeEvent.Oid,
                Path:  completeEvent.Path,
            })
            if err != nil {
                log.Fatalf("Failed to marshal response: %v", err)
            }
            // log.Printf("Sending response without error: %s", string(jsonResponse)) // Add this line for logging
            fmt.Println(string(jsonResponse))
            return
        }
    }

    jsonResponse, err := json.Marshal(response)
    if err != nil {
        log.Fatalf("Failed to marshal response: %v", err)
    }
    log.Printf("Sending response with error: %s", string(jsonResponse)) // Add this line for logging
    fmt.Println(string(jsonResponse))
}

func main() {
    
    // send all debugging log messages to Stderr so they don't interfere with response
    log.SetOutput(os.Stderr)
    
    scanner := bufio.NewScanner(os.Stdin)

    sess := session.Must(session.NewSession(&aws.Config{
        Region:      aws.String(os.Getenv("LFS_AWS_REGION")),
        Endpoint:    aws.String(os.Getenv("LFS_AWS_ENDPOINT")),
        Credentials: credentials.NewSharedCredentials("", os.Getenv("LFS_AWS_PROFILE")),
    }))
    svc := s3.New(sess)

    for scanner.Scan() {
        var event map[string]interface{}
        json.Unmarshal(scanner.Bytes(), &event)

        switch event["event"] {
        case "init":
            var initEvent InitEvent
            json.Unmarshal(scanner.Bytes(), &initEvent)
            handleInit(initEvent)
        case "upload":
            var uploadEvent TransferEvent
            json.Unmarshal(scanner.Bytes(), &uploadEvent)
            handleUpload(uploadEvent, svc)
        case "download":
            var downloadEvent TransferEvent
            json.Unmarshal(scanner.Bytes(), &downloadEvent)
            handleDownload(downloadEvent, svc)
        case "terminate":
            return
        }
    }

    if err := scanner.Err(); err != nil {
        log.Fatalf("Error reading standard input: %v", err)
    }
}
