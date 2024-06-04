package main

import (
    "crypto/sha256"
    "encoding/hex"
    "bufio"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "os"
    "path/filepath"

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

    // Set log output to stdout
    log.SetOutput(os.Stdout)

    // Log the loggingEnabled status
    // log.Printf("loggingEnabled %t \n", loggingEnabled)
    
    // Compute local file's SHA-256 hash
    localSHA256, err := computeSHA256(event.Path)
    if err != nil {
        response := CompleteEvent{
            Event: "complete",
            Oid:   event.Oid,
            Error: struct {
                Code    int    `json:"code"`
                Message string `json:"message"`
            }{
                Code:    1,
                Message: fmt.Sprintf("Failed to compute SHA-256 hash of file %q: %v", event.Path, err),
            },
        }
        sendResponse(response)
        return
    }
    
    // Check if the object already exists
    key := fmt.Sprintf("%s/%s", os.Getenv("LFS_AWS_USER"), event.Oid)
    headObjInput := &s3.HeadObjectInput{
        Bucket: aws.String(os.Getenv("LFS_S3_BUCKET")),
        Key:    aws.String(key),
    }

    headObjOutput, err := svc.HeadObject(headObjInput)
    if err == nil {
        // Get the SHA-256 hash from metadata
        remoteSHA256 := headObjOutput.Metadata["Sha256"]

        // Compare hashes
        if remoteSHA256 != nil && *remoteSHA256 == localSHA256 {
            if loggingEnabled {
                log.Printf("oid %s already exists at key %s with matching SHA-256, skipping upload\n", event.Oid, key)
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
        response := CompleteEvent{
            Event: "complete",
            Oid:   event.Oid,
            Error: struct {
                Code    int    `json:"code"`
                Message string `json:"message"`
            }{
                Code:    1,
                Message: fmt.Sprintf("Failed to open file %q: %v", event.Path, err),
            },
        }
        sendResponse(response)
        return
    }
    defer file.Close()

    // Compute SHA-256 hash for the upload
    // sha256Hash, err := computeSHA256(event.Path)
    if err != nil {
        response := CompleteEvent{
            Event: "complete",
            Oid:   event.Oid,
            Error: struct {
                Code    int    `json:"code"`
                Message string `json:"message"`
            }{
                Code:    1,
                Message: fmt.Sprintf("Failed to compute SHA-256 hash of file %q: %v", event.Path, err),
            },
        }
        sendResponse(response)
        return
    }

    _, err = svc.PutObject(&s3.PutObjectInput{
        Bucket: aws.String(os.Getenv("LFS_S3_BUCKET")),
        Key:    aws.String(key),
        Body:   file,
        Metadata: map[string]*string{
            "Sha256": aws.String(localSHA256),
        },
    })
    if err != nil {
        response := CompleteEvent{
            Event: "complete",
            Oid:   event.Oid,
            Error: struct {
                Code    int    `json:"code"`
                Message string `json:"message"`
            }{
                Code:    1,
                Message: fmt.Sprintf("Failed to upload data to %s/%s: %v", os.Getenv("LFS_S3_BUCKET"), key, err),
            },
        }
        sendResponse(response)
        return
    }

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

func handleDownload(event TransferEvent, svc *s3.S3) {
    loggingEnabled := os.Getenv("LFS_LOGGING") == "true"
    
    // Set log output to stdout
    log.SetOutput(os.Stdout)
    
    localStorage := os.Getenv("LFS_LOCAL_STORAGE")
    if localStorage == "" {
        localStorage = defaultLfsStorage
    }

    // Compute the file path for checking the S3 object
    key := fmt.Sprintf("%s/%s", os.Getenv("LFS_AWS_USER"), event.Oid)
    headObjOutput, err := svc.HeadObject(&s3.HeadObjectInput{
        Bucket: aws.String(os.Getenv("LFS_S3_BUCKET")),
        Key:    aws.String(key),
    })
    if err != nil {
        handleError(event, fmt.Errorf("failed to head data from %s/%s: %v", os.Getenv("LFS_S3_BUCKET"), key, err))
        return
    }

    // Get the SHA-256 hash from metadata
    remoteSHA256 := headObjOutput.Metadata["Sha256"]
    localPath := ""

    if remoteSHA256 == nil {
        // Download the file to a temporary location
        tmpPath := filepath.Join("/tmp", event.Oid)
        err := downloadFileFromS3(svc, os.Getenv("LFS_S3_BUCKET"), key, tmpPath)
        if err != nil {
            handleError(event, err)
            return
        }

        // Compute the SHA-256 hash of the downloaded file
        localSHA256, err := computeSHA256(tmpPath)
        if err != nil {
            handleError(event, fmt.Errorf("failed to compute SHA-256 hash of file %q: %v", tmpPath, err))
            return
        }

        // Move the file to the correct local path
        localPath = filepath.Join(localStorage, localSHA256, event.Oid)
        err = os.MkdirAll(filepath.Dir(localPath), os.ModePerm)
        if err != nil {
            handleError(event, fmt.Errorf("failed to create directory for file %q: %v", localPath, err))
            return
        }

        err = os.Rename(tmpPath, localPath)
        if err != nil {
            handleError(event, fmt.Errorf("failed to move file from %q to %q: %v", tmpPath, localPath, err))
            return
        }

        // Update the S3 metadata with the computed SHA-256 hash
        err = updateS3ObjectMetadata(svc, os.Getenv("LFS_S3_BUCKET"), key, localSHA256)
        if err != nil {
            handleError(event, err)
            return
        }

    } else {
        // If the remote SHA-256 is not nil, compare it with the local file's hash
        localPath = filepath.Join(localStorage, *remoteSHA256, event.Oid)
        localFileInfo, err := os.Stat(localPath)
        if err == nil && localFileInfo.Size() == event.Size {
            // Compute the SHA-256 hash of the local file
            localSHA256, err := computeSHA256(localPath)
            if err == nil && *remoteSHA256 == localSHA256 {
                if loggingEnabled {
                    log.Printf("oid %s already exists at localPath %s, skipping download\n", event.Oid, localPath)
                }                
                // The local file exists, sizes match, and the hashes match, so skip the download
                response := CompleteEvent{Event: "complete", Oid: event.Oid, Path: localPath}
                sendResponse(response)
                return
            }
        }

        // Proceed to download the file
        err = downloadFileFromS3(svc, os.Getenv("LFS_S3_BUCKET"), key, localPath)
        if err != nil {
            handleError(event, err)
            return
        }
    }

    response := CompleteEvent{Event: "complete", Oid: event.Oid, Path: localPath}
    sendResponse(response)
}

// func handleDownload(event TransferEvent, svc *s3.S3) {
//     loggingEnabled := os.Getenv("LFS_LOGGING") == "true"
    
//     // Set log output to stdout
//     log.SetOutput(os.Stdout)
    
//     localStorage := os.Getenv("LFS_LOCAL_STORAGE")
//     if localStorage == "" {
//         localStorage = defaultLfsStorage
//     }

//     localPath := filepath.Join(localStorage, event.Oid)

//     // Check if the local file exists and its size
//     localFileInfo, err := os.Stat(localPath)
//     if err == nil && localFileInfo.Size() == event.Size {
//         if loggingEnabled {
//             log.Printf("oid %s already exists at localPath %s, skipping download\n", event.Oid, localPath)
//         }
//         // Local file exists and sizes match, skip download
//         response := CompleteEvent{Event: "complete", Oid: event.Oid, Path: localPath}
//         sendResponse(response)
//         return
//     }

//     // Proceed to download the object
//     file, err := os.Create(localPath)
//     if err != nil {
//         response := CompleteEvent{
//             Event: "complete",
//             Oid:   event.Oid,
//             Error: struct {
//                 Code    int    `json:"code"`
//                 Message string `json:"message"`
//             }{
//                 Code:    1,
//                 Message: fmt.Sprintf("Failed to create file %q: %v", localPath, err),
//             },
//         }
//         sendResponse(response)
//         return
//     }
//     defer file.Close()

//     key := fmt.Sprintf("%s/%s", os.Getenv("LFS_AWS_USER"), event.Oid)
//     output, err := svc.GetObject(&s3.GetObjectInput{
//         Bucket: aws.String(os.Getenv("LFS_S3_BUCKET")),
//         Key:    aws.String(key),
//     })
//     if err != nil {
//         response := CompleteEvent{
//             Event: "complete",
//             Oid:   event.Oid,
//             Error: struct {
//                 Code    int    `json:"code"`
//                 Message string `json:"message"`
//             }{
//                 Code:    1,
//                 Message: fmt.Sprintf("Failed to download data from %s/%s: %v", os.Getenv("LFS_S3_BUCKET"), key, err),
//             },
//         }
//         sendResponse(response)
//         return
//     }
//     defer output.Body.Close()

//     _, err = io.Copy(file, output.Body)
//     if err != nil {
//         response := CompleteEvent{
//             Event: "complete",
//             Oid:   event.Oid,
//             Error: struct {
//                 Code    int    `json:"code"`
//                 Message string `json:"message"`
//             }{
//                 Code:    1,
//                 Message: fmt.Sprintf("Failed to write data to file %q: %v", localPath, err),
//             },
//         }
//         sendResponse(response)
//         return
//     }

//     response := CompleteEvent{Event: "complete", Oid: event.Oid, Path: localPath}
//     sendResponse(response)
// }

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
            fmt.Println(string(jsonResponse))
            return
        }
    }

    jsonResponse, err := json.Marshal(response)
    if err != nil {
        log.Fatalf("Failed to marshal response: %v", err)
    }
    fmt.Println(string(jsonResponse))
}

func main() {
    
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
