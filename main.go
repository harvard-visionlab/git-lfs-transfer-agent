package main

import (
    "encoding/json"
    "fmt"
    "io"
    "log"
    "os"
)

type TransferRequest struct {
    Event      string `json:"event"`
    Operation  string `json:"operation"`
    Remote     string `json:"remote"`
    Concurrent bool   `json:"concurrent"`
    Objects    []struct {
        Oid  string `json:"oid"`
        Size int64  `json:"size"`
    } `json:"objects"`
}

type TransferResponse struct {
    Objects []struct {
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
    } `json:"objects"`
}

func main() {
    log.Println("Starting lfs-transfer-agent")
    decoder := json.NewDecoder(os.Stdin)
    encoder := json.NewEncoder(os.Stdout)

    for {
        var req TransferRequest
        if err := decoder.Decode(&req); err == io.EOF {
            break
        } else if err != nil {
            log.Fatalf("Error decoding request: %v", err)
        }

        log.Printf("Received request: %+v", req)

        var res TransferResponse
        for _, obj := range req.Objects {
            var actionUrl string
            actions := struct {
                Download struct {
                    Href   string            `json:"href"`
                    Header map[string]string `json:"header"`
                } `json:"download,omitempty"`
                Upload struct {
                    Href   string            `json:"href"`
                    Header map[string]string `json:"header"`
                } `json:"upload,omitempty"`
            }{}

            actionUrl = fmt.Sprintf("https://your-lambda-function-url?oid=%s", obj.Oid)
            actions.Download.Href = actionUrl
            actions.Download.Header = map[string]string{
                "x-api-key": os.Getenv("LFS_API_KEY"),
            }
            actions.Upload.Href = actionUrl
            actions.Upload.Header = map[string]string{
                "x-api-key": os.Getenv("LFS_API_KEY"),
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
                Oid:     obj.Oid,
                Actions: actions,
            })
        }

        if err := encoder.Encode(&res); err != nil {
            log.Fatalf("Error encoding response: %v", err)
        }

        log.Printf("Sent response: %+v", res)
    }

    log.Println("lfs-transfer-agent finished")
}
