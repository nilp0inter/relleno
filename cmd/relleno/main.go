package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

type State struct {
	Url          string `json:"url"`
	Method       string `json:"method"`
	Delete       bool   `json:"delete"`
	SendDocument bool   `json:"sendDocument"`
}

type Task struct {
	Document json.RawMessage  `json:"document"`
	Schema   json.RawMessage  `json:"schema"`
	Spa      string           `json:"spa"`
	State    string           `json:"state"`
	Config   json.RawMessage  `json:"config"`
	States   map[string]State `json:"states"`
}

type config struct {
	docPath string
}

func validateTask(r io.Reader) (Task, error) {
	var d Task

	err := json.NewDecoder(r).Decode(&d)
	if err != nil {
		return d, err
	}

	// Load "schema", implicitly validating it
	sch, err := jsonschema.CompileString("schema", string(d.Schema))
	if err != nil {
		return d, err
	}

	var v interface{}
	if err := json.Unmarshal([]byte(d.Document), &v); err != nil {
		return d, err
	}

	// Validate "document" against "schema"
	if err = sch.Validate(v); err != nil {
		return d, err
	}

	return d, nil
}

func (c config) deleteTask(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	err := os.Remove(filepath.Join(c.docPath, vars["id"]))
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c config) updateDocument(w http.ResponseWriter, req *http.Request) {
	var d Task
	vars := mux.Vars(req)
	// Open our jsonFile
	jsonFile, err := os.Open(filepath.Join(c.docPath, vars["id"]))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer jsonFile.Close()
	err = json.NewDecoder(jsonFile).Decode(&d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewDecoder(req.Body).Decode(&d.Document)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Load "schema", implicitly validating it
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

	// Validate "document" against "schema"
	if err = sch.Validate(v); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	// Update state
	file, _ := json.MarshalIndent(d, "", " ")
	err = ioutil.WriteFile(filepath.Join(c.docPath, vars["id"]), file, 0644)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c config) getConfig(w http.ResponseWriter, req *http.Request) {
	var d Task
	vars := mux.Vars(req)
	// Open our jsonFile
	jsonFile, err := os.Open(filepath.Join(c.docPath, vars["id"]))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer jsonFile.Close()
	err = json.NewDecoder(jsonFile).Decode(&d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(d.Config)
}

func (c config) getSpa(w http.ResponseWriter, req *http.Request) {
	var d Task
	vars := mux.Vars(req)
	// Open our jsonFile
	jsonFile, err := os.Open(filepath.Join(c.docPath, vars["id"]))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer jsonFile.Close()
	err = json.NewDecoder(jsonFile).Decode(&d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	// TODO: serve default SPA if is not defined
	io.WriteString(w, d.Spa)
}

func (c config) getSchema(w http.ResponseWriter, req *http.Request) {
	var d Task
	vars := mux.Vars(req)
	// Open our jsonFile
	jsonFile, err := os.Open(filepath.Join(c.docPath, vars["id"]))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer jsonFile.Close()
	err = json.NewDecoder(jsonFile).Decode(&d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, string(d.Schema))
}

func (c config) getDocument(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	// Open our jsonFile
	jsonFile, err := os.Open(filepath.Join(c.docPath, vars["id"]))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer jsonFile.Close()
	d, err := validateTask(jsonFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, string(d.Document))
}

func (c config) createTask(w http.ResponseWriter, req *http.Request) {
	var d Task

	d, err := validateTask(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	id := uuid.NewString()

	file, _ := json.MarshalIndent(d, "", " ")
	err = ioutil.WriteFile(filepath.Join(c.docPath, id), file, 0644)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, id)
}

func (c config) getState(w http.ResponseWriter, req *http.Request) {
	var d Task
	vars := mux.Vars(req)
	// Open our jsonFile
	jsonFile, err := os.Open(filepath.Join(c.docPath, vars["id"]))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer jsonFile.Close()
	err = json.NewDecoder(jsonFile).Decode(&d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(d.State)
}

func (c config) changeState(w http.ResponseWriter, req *http.Request) {
	var d Task
	vars := mux.Vars(req)
	// Open our jsonFile
	jsonFile, err := os.Open(filepath.Join(c.docPath, vars["id"]))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer jsonFile.Close()
	err = json.NewDecoder(jsonFile).Decode(&d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewDecoder(req.Body).Decode(&d.State)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	action, ok := d.States[d.State]
	if !ok {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	if action.Url != "" {
		// Make request
		fmt.Printf("Sending request to %s %s\n", action.Method, action.Url)
	}
	if action.Delete {
		err := os.Remove(filepath.Join(c.docPath, vars["id"]))
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Update state
		file, _ := json.MarshalIndent(d, "", " ")
		err = ioutil.WriteFile(filepath.Join(c.docPath, vars["id"]), file, 0644)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func main() {
	var dir string

	flag.StringVar(&dir, "dir", ".", "the directory to serve files from. Defaults to the current dir")
	flag.Parse()

	c := config{docPath: dir}

	r := mux.NewRouter()
	r.HandleFunc("/doc", c.createTask).Methods("POST")
	r.HandleFunc("/doc/{id:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}", c.getSpa).Methods("GET")
	r.HandleFunc("/doc/{id:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}", c.deleteTask).Methods("DELETE")
	r.HandleFunc("/doc/{id:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}/document", c.getDocument).Methods("GET")
	r.HandleFunc("/doc/{id:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}/document", c.updateDocument).Methods("PUT")
	r.HandleFunc("/doc/{id:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}/schema", c.getSchema).Methods("GET")
	r.HandleFunc("/doc/{id:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}/config", c.getConfig).Methods("GET")
	r.HandleFunc("/doc/{id:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}/state", c.getState).Methods("GET")
	r.HandleFunc("/doc/{id:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}/state", c.changeState).Methods("POST")

	log.Println("Listing for requests at http://localhost:8001/")
	srv := &http.Server{
		Handler: logRequest(r),
		Addr:    "127.0.0.1:8001",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
