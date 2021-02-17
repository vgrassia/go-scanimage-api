package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
)

var addr string
var device string
var port int

func init() {
	const (
		defaultAddr = ""
		defaultPort = 8080
		addrUsage   = "address for webserver to listen on"
		portUsage   = "port for webserver to listen on"

		defaultDevice = ""
		deviceUsage   = "device used for scanning"
	)
	flag.StringVar(&addr, "address", defaultAddr, addrUsage)
	flag.StringVar(&addr, "a", defaultAddr, addrUsage+" (shorthand)")
	flag.IntVar(&port, "port", defaultPort, portUsage)
	flag.IntVar(&port, "p", defaultPort, portUsage+" (shorthand)")

	flag.StringVar(&device, "device", defaultDevice, deviceUsage)
	flag.StringVar(&device, "d", defaultDevice, deviceUsage+" (shorthand)")
}

func main() {
	// Parse command line flags and validate them.
	flag.Parse()
	if device == "" {
		log.Fatalln("You must specify a device to scan from.")
	}

	log.Printf("Starting webserver on %s:%d...", addr, port)
	http.HandleFunc("/scan", handleScan)
	log.Fatalln(http.ListenAndServe(fmt.Sprintf("%s:%d", addr, port), nil))
}

func handleScan(w http.ResponseWriter, r *http.Request) {
	// We don't allow any requests except POST to this endpoint.
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Parse form grabs the form values from the request.
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	// Scan defaults
	format := "png"
	mode := "gray"
	resolution := "300"

	// Set scan parameters from POST data.
	if reqFormat := r.PostForm.Get("format"); reqFormat != "" {
		format = reqFormat
	}
	if reqMode := r.PostForm.Get("mode"); reqMode != "" {
		mode = reqMode
	}
	if reqResolution := r.PostForm.Get("resolution"); reqResolution != "" {
		resolution = reqResolution
	}

	// Create command.
	cmd := exec.Command(
		"scanimage",
		fmt.Sprintf("--device-name=%s", device),
		fmt.Sprintf("--format=%s", format),
		fmt.Sprintf("--mode=%s", mode),
		fmt.Sprintf("--resolution=%s", resolution),
	)

	// Set command's stdout and stderr.
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		w.WriteHeader(http.StatusGatewayTimeout)
		log.Printf("Err: %s | %s", err, stderr.String())
		if err.Error() == "exit status 7" {
			w.Write([]byte("Document feeder out of documents"))
		}
		return
	}

	// Read bytes from stdout and send response.
	reader := bytes.NewReader(stdout.Bytes())
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	response := struct {
		Data []byte
	}{
		Data: data,
	}
	responseJSON, err := json.Marshal(&response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	w.Write(responseJSON)
}
