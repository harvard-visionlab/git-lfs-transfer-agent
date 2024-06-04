package main

import (
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

    key := fmt.Sprintf("%s/%s", os.Getenv("LFS_AWS_USER"), event.Oid)
    _, err = svc.PutObject(&s3.PutObjectInput{
        Bucket: aws.String(os.Getenv("LFS_S3_BUCKET")),
        Key:    aws.String(key),
        Body:   file,
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

func handleDownload(event TransferEvent, svc *s3.S3) {
    localStorage := os.Getenv("LFS_LOCAL_STORAGE")
    if localStorage == "" {
        localStorage = defaultLfsStorage
    }

    localPath := filepath.Join(localStorage, event.Oid)
    file, err := os.Create(localPath)
    if err != nil {
        response := CompleteEvent{
            Event: "complete",
            Oid:   event.Oid,
            Error: struct {
                Code    int    `json:"code"`
                Message string `json:"message"`
            }{
                Code:    1,
                Message: fmt.Sprintf("Failed to create file %q: %v", localPath, err),
            },
        }
        sendResponse(response)
        return
    }
    defer file.Close()

    key := fmt.Sprintf("%s/%s", os.Getenv("LFS_AWS_USER"), event.Oid)
    output, err := svc.GetObject(&s3.GetObjectInput{
        Bucket: aws.String(os.Getenv("LFS_S3_BUCKET")),
        Key:    aws.String(key),
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
                Message: fmt.Sprintf("Failed to download data from %s/%s: %v", os.Getenv("LFS_S3_BUCKET"), key, err),
            },
        }
        sendResponse(response)
        return
    }
    defer output.Body.Close()

    _, err = io.Copy(file, output.Body)
    if err != nil {
        response := CompleteEvent{
            Event: "complete",
            Oid:   event.Oid,
            Error: struct {
                Code    int    `json:"code"`
                Message string `json:"message"`
            }{
                Code:    1,
                Message: fmt.Sprintf("Failed to write data to file %q: %v", localPath, err),
            },
        }
        sendResponse(response)
        return
    }

    response := CompleteEvent{Event: "complete", Oid: event.Oid, Path: localPath}
    sendResponse(response)
}

// func sendResponse(response interface{}) {
//     jsonResponse, err := json.Marshal(response)
//     if err != nil {
//         log.Fatalf("Failed to marshal response: %v", err)
//     }
//     fmt.Println(string(jsonResponse))
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
