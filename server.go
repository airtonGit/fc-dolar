package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	urlCotacao = "https://economia.awesomeapi.com.br/json/last/USD-BRL"
	dbFile     = "fc-dolar.db"

	apiTimeout = 200
	dbTimeout  = 10
)

func main() {
	http.HandleFunc("/cotacao", handler)
	println("Server listening...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type Cotacao struct {
	USDBRL struct {
		Bid string `json:"bid"`
	} `json:"USDBRL"`
}

type Repository struct {
	DB *sql.DB
}

const insertCotacao = `INSERT INTO cotacao (created_at, bid) VALUES (?, ?);`

var (
	ErrInsertCotacaoPrepare = errors.New("insert_cotacao_prepare")
	ErrInsertCotacaoExec    = errors.New("insert_cotacao_exec")
)

func (r *Repository) InsertCotacao(ctx context.Context, bid string) error {
	stmt, err := r.DB.PrepareContext(ctx, insertCotacao)
	if err != nil {
		return errors.Join(ErrInsertCotacaoPrepare, fmt.Errorf("insert cotacao prepare %v", err))
	}
	_, err = stmt.ExecContext(ctx, time.Now(), bid)
	if err != nil {
		return errors.Join(ErrInsertCotacaoExec, fmt.Errorf("insert cotacao exec %v", err))
	}
	return nil
}

func internalServerError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	apiRequestCtx, cancelApiRequest := context.WithTimeout(ctx, apiTimeout*time.Millisecond)
	defer cancelApiRequest()

	cotacao, err := exchangeAPIRequest(apiRequestCtx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			w.WriteHeader(http.StatusRequestTimeout)
			w.Write([]byte("request timeout"))
			log.Printf("request timeout %v", err)
			return
		}

		internalServerError(w, err)
		log.Printf("fail %v", err)
		return
	}

	// Repository Store
	err = dbStore(ctx, cotacao)
	if err != nil {
		internalServerError(w, err)
		log.Printf("fail %v", err)
	}

	type cotacaoResponse struct {
		Bid string `json:"bid"`
	}

	cotacaoResp := cotacaoResponse{
		Bid: cotacao.USDBRL.Bid,
	}
	err = json.NewEncoder(w).Encode(cotacaoResp)
	if err != nil {
		internalServerError(w, err)
		log.Printf("fail %v", err)
	}
}

func dbStore(ctx context.Context, cotacao Cotacao) error {
	conn, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return err
	}

	repo := Repository{DB: conn}
	dbStoreCtx, cancelDBStore := context.WithTimeout(ctx, dbTimeout*time.Millisecond)
	defer cancelDBStore()

	err = repo.InsertCotacao(dbStoreCtx, cotacao.USDBRL.Bid)
	if err != nil {
		return err
	}
	return nil
}

func exchangeAPIRequest(ctx context.Context) (Cotacao, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlCotacao, nil)
	if err != nil {
		panic(err.Error())
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Cotacao{}, err
	}

	var cotacao Cotacao
	err = json.NewDecoder(resp.Body).Decode(&cotacao)
	if err != nil {
		return Cotacao{}, err
	}
	return cotacao, err
}
