package main

import (
    "encoding/json"
    "fmt"
    "io"
    "log"
    "os"
)

type TransferRequest struct {
    Operation string `json:"operation"`
    Objects   []struct {
        Oid  string `json:"oid"`
        Size int64  `json:"size"`
    } `json:"objects"`
}

type TransferResponse struct {
    Objects []struct {
        Oid  string `json:"oid"`
        Actions struct {
            Download struct {
                Href string `json:"href"`
                Header map[string]string `json:"header"`
            } `json:"download,omitempty"`
            Upload struct {
                Href string `json:"href"`
                Header map[string]string `json:"header"`
            } `json:"upload,omitempty"`
        } `json:"actions"`
    } `json:"objects"`
}

func main() {
    decoder := json.NewDecoder(os.Stdin)
    encoder := json.NewEncoder(os.Stdout)
    
    // Get the base URL from the environment variable
    baseURL := os.Getenv("LFS_LAMBDA_FUNCTION_URL")
    if baseURL == "" {
        log.Fatalf("Environment variable LAMBDA_FUNCTION_URL is not set.")
    }
    
    for {
        var req TransferRequest
        if err := decoder.Decode(&req); err == io.EOF {
            break
        } else if err != nil {
            log.Fatalf("Error decoding request: %v", err)
        }

        var res TransferResponse
        for _, obj := range req.Objects {
            var actionUrl string
            if req.Operation == "download" {
                actionUrl = fmt.Sprintf("https://%s?oid=%s", baseURL, obj.Oid)
            } else if req.Operation == "upload" {
                actionUrl = fmt.Sprintf("https://%s?oid=%s", baseURL, obj.Oid)
            }

            res.Objects = append(res.Objects, struct {
                Oid     string `json:"oid"`
                Actions struct {
                    Download struct {
                        Href   string            `json:"href"`
                        Header map[string]string `json:"header"`
                    } `json:"download,omitempty"`
                    Upload struct {
                        Href   string            `json:"href"`
                        Header map[string]string `json:"header"`
                    } `json:"upload,omitempty"`
                } `json:"actions"`
            }{
                Oid: obj.Oid,
                Actions: struct {
                    Download struct {
                        Href   string            `json:"href"`
                        Header map[string]string `json:"header"`
                    } `json:"download,omitempty"`
                    Upload struct {
                        Href   string            `json:"href"`
                        Header map[string]string `json:"header"`
                    } `json:"upload,omitempty"`
                }{
                    Download: struct {
                        Href   string            `json:"href"`
                        Header map[string]string `json:"header"`
                    }{
                        Href: actionUrl,
                        Header: map[string]string{
                            "x-api-key": os.Getenv("LFS_API_KEY"),
                        },
                    },
                    Upload: struct {
                        Href   string            `json:"href"`
                        Header map[string]string `json:"header"`
                    }{
                        Href: actionUrl,
                        Header: map[string]string{
                            "x-api-key": os.Getenv("LFS_API_KEY"),
                        },
                    },
                },
            })
        }

        if err := encoder.Encode(&res); err != nil {
            log.Fatalf("Error encoding response: %v", err)
        }
    }
}
