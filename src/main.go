package main

import (
    "gopkg.in/alecthomas/kingpin.v2"
    "net/http"
    "log"
    "time"
    "github.com/gorilla/mux"
    "github.com/gorilla/handlers"
    "os"
    "fmt"
    "crypto/md5"
    "encoding/hex"
    "io/ioutil"
    "strconv"
    "io"
)

var (
    cachePath = kingpin.Flag("cache-path", "Cache path to store the downloaded files").Required().String()
    host = kingpin.Flag("host", "Host").Default("0.0.0.0").String()
    port = kingpin.Flag("port", "Port").Default("80").Int()
)

func main() {
    kingpin.Parse()
    router := mux.NewRouter()
    router.HandleFunc("/{name:.*}", handleRequest)
    loggedRouter := handlers.LoggingHandler(os.Stdout, router)
    log.Printf("Serving at %s:%d\n", *host, *port)
    log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", *host, *port), loggedRouter))
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
    // Get the file from the headers
    file := r.Header.Get("File")
    if len(file) == 0 {
        returnError(400, "File header not found", w)
        return
    }

    // Get the last modified for the file
    lastModified, err := getRemoteFileLastModified(file)
    if err != nil {
        returnError(400, err, w)
        return
    }

    // Get the last modified for the local file
    fileHash := getMD5Hash(file)
    localLastModified, err := getLocalFileLastModified(fileHash)
    if err != nil {
        returnError(500, err, w)
        return
    }

    // Download the file to local if it doesn't already exist
    if lastModified != localLastModified {
        if err = getRemoteFile(file, fileHash); err != nil {
            returnError(500, err, w)
            return
        }
        if err = updateFileIndex(fileHash, lastModified); err != nil {
            returnError(500, err, w)
            return
        }
    }

    // Serve from the local file
    b, err := ioutil.ReadFile(*cachePath + fileHash)
    if err != nil {
        returnError(500, err , w)
        return
    }

    w.WriteHeader(200)
    w.Write(b)
    return
}

func getRemoteFileLastModified(url string) (int64, error) {
    resp, err := http.Head(url)
    if err != nil {
        return 0, err
    }
    lastModified := resp.Header.Get("Last-Modified")
    if len(lastModified) == 0 {
        return 0, nil
    }
    // Last Modified header is in RFC1123 format
    parsedTime, err := time.Parse(time.RFC1123, lastModified)
    return parsedTime.Unix(), err
}

func getLocalFileLastModified(hash string) (int64, error) {
    b, err := ioutil.ReadFile(*cachePath + hash + ".index")
    if err != nil {
        return 0, nil
    }
    return strconv.ParseInt(string(b), 0, 64)
}

func getMD5Hash(text string) string {
    hasher := md5.New()
    hasher.Write([]byte(text))
    return hex.EncodeToString(hasher.Sum(nil))
}

func getRemoteFile(url string, hash string) error {
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    out, err := os.Create(*cachePath + hash)
    if err != nil {
        return err
    }
    defer out.Close()
    _, err = io.Copy(out, resp.Body)
    return err
}

func updateFileIndex(hash string, lastModified int64) error {
    out, err := os.Create(*cachePath + hash + ".index")
    if err != nil {
        return err
    }
    defer out.Close()
    _, err = io.WriteString(out, strconv.FormatInt(lastModified, 10))
    return err
}

func returnError(code int, err interface{}, w http.ResponseWriter) {
    w.WriteHeader(code)
    log.Println(err)
}
