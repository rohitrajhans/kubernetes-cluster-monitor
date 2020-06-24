package main

// A simple server that stores the logs received from the logger via push mechanism

import (
	"fmt"
	"os"
	"net/http"
	"time"
	// "strconv"
	"encoding/json"
	"encoding/csv"
	// "io/ioutil"
	// "strings"
	"log"

	"github.com/gorilla/mux"
)

// struct to store log info
// similar to sidecar/sidecar.go
type loginfo struct {
	Time 		string	`json:"time"`
	Level 		string	`json:"level"`
	Msg 		string	`json:"msg"`
	Container 	string	`json:"container"`
	Pod 		string	`json:"pod"`
	Request 	string	`json:"request"`
	Status 		string	`json:"status"`
	StatusCode 	string	`json:"statuscode"`
	Msgtype 	string	`json:"type"`
	Url			string	`json:"url"`
}

// function to create new router
func newRouter() *mux.Router {

	r := mux.NewRouter()
	r.HandleFunc("/", responseMain).Methods("GET")
	r.HandleFunc("/", receiveLogs).Methods("POST")
	
	return r
}

func responseMain(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "/get_logs")
}

// stores received logs in `received.csv`
func receiveLogs(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Fprintf(w, "Parseform() err: %v", err)
		return
	}

	data := r.FormValue("data")
	var logdata []loginfo
	err = json.Unmarshal([]byte(data), &logdata)
	if err != nil {
		fmt.Fprintf(w, "Decode err: %v", err)
		return
	}
	// fmt.Println(logdata)

	file, err := os.OpenFile("received.csv",  os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		fmt.Println("Unable to open file")
	}
	defer file.Close()
	wr := csv.NewWriter(file)
	for _, obj := range logdata {
		var record []string
		record = append(record, obj.Time)
		record = append(record, obj.Level)
		record = append(record, obj.Msg)
		record = append(record, obj.Container)
		record = append(record, obj.Pod)
		record = append(record, obj.Request)
		record = append(record, obj.Status)
		record = append(record, obj.StatusCode)
		record = append(record, obj.Msgtype)
		record = append(record, obj.Url)
		wr.Write(record)
	}
	
	tm := time.Now()
	fmt.Printf("%s", tm.Format("2006-01-02 15:04:05"))
	fmt.Printf(" : Wrote %d bytes to file\n", len(data))
	wr.Flush()
	fmt.Fprintf(w, "Data received")
}

func main() {
	// serving on port: 3000
	port := ":3000"
	r := newRouter()
	fmt.Printf("Now serving on Port %s\n", port)
	log.Fatal(http.ListenAndServe(port, r))
}
