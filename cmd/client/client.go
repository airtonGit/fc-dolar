package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	requestTimeout  = 3000
	urlRequest      = "http://127.0.0.1:8080/cotacao"
	cotacaoFilename = "cotacao.txt"
)

type CotacaoPayload struct {
	Bid string `json:"bid"`
}

func main() {

	body := makeRequest()

	cotacaoPayload := decodePayload(body)

	writeFile(cotacaoPayload)
}

func writeFile(cotacaoPayload CotacaoPayload) {
	cotacaoFile, err := os.OpenFile(cotacaoFilename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		panic(err)
	}
	defer cotacaoFile.Close()
	cotacaoFile.WriteString(fmt.Sprintf("Dólar: %s", cotacaoPayload.Bid))
}

func decodePayload(payload io.ReadCloser) CotacaoPayload {

	var cotacaoPayload CotacaoPayload

	err := json.NewDecoder(payload).Decode(&cotacaoPayload)
	if err != nil {
		panic(err)
	}
	return cotacaoPayload
}

func makeRequest() io.ReadCloser {
	client := http.Client{}
	client.Timeout = requestTimeout * time.Millisecond

	req, err := http.NewRequest(http.MethodGet, urlRequest, nil)
	if err != nil {
		panic(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode > http.StatusAccepted {
		log.Fatalf("response error %d", resp.StatusCode)
	}

	return resp.Body
}
