package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/santhosh-tekuri/jsonschema/v5"
	_ "github.com/santhosh-tekuri/jsonschema/v5/httploader"
)

type Doc struct {
	Document json.RawMessage
	Schema   json.RawMessage
}

func listDocs(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "Hello, world!\n")
}

func postDoc(w http.ResponseWriter, req *http.Request) {
	var d Doc

	// Try to decode the request body into the struct. If there is an error,
	// respond to the client with the error message and a 400 status code.
	err := json.NewDecoder(req.Body).Decode(&d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sch, err := jsonschema.CompileString("schema", string(d.Schema))
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	var v interface{}
	if err := json.Unmarshal([]byte(d.Document), &v); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err = sch.Validate(v); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.WriteHeader(http.StatusCreated)
	// Do something with the Person struct...
	fmt.Fprintf(w, ";)")
}

func main() {
	// Hello world, the web server
	r := mux.NewRouter()
	r.HandleFunc("/docs", listDocs).Methods("GET")
	r.HandleFunc("/docs", postDoc).Methods("POST")

	log.Println("Listing for requests at http://localhost:8000/")
	srv := &http.Server{
		Handler: r,
		Addr:    "127.0.0.1:8000",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
