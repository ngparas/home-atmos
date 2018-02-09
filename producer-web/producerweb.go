package main

import (
    "encoding/json"
    //"errors"
    "fmt"
    "io"
    "io/ioutil"
    "net/http"
    "os"
)

var (
    PORT string = os.Getenv("PORT")
)

type SensorValue struct {
    DeviceId string
    SignalTime int
    SignalName string
    SignalValue float64
}

func isJsonContent(h http.Header) bool {
    contentType, ok := h["Content-Type"]
    return ok && len(contentType)==1 && contentType[0] == "application/json"
}

func (sv *SensorValue) LoadContent(body io.ReadCloser) error {
    bodyBytes, err := ioutil.ReadAll(body)
    if err != nil {
        return err
    }

    err = json.Unmarshal(bodyBytes, sv)
    if err != nil {
        return err
    }
    return nil
}

func baseHandler(w http.ResponseWriter, r *http.Request) {

    if r.Method == "POST" {

        if isJsonContent(r.Header) {

            var j SensorValue
            err := j.LoadContent(r.Body)
            if err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }

            fmt.Println(j)
            fmt.Fprintf(w, "OK")

        } else {
            http.Error(w, "Content-type must be application/json", http.StatusInternalServerError)
            return
        }

    } else {
        http.Error(w, "Only POST Requests are allowed", http.StatusInternalServerError)
        return
    }

}

func main() {
    http.HandleFunc("/", baseHandler)
    http.ListenAndServe(":"+PORT, nil)

}
