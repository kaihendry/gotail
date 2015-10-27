package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ActiveState/tail"
)

var filename string

type Broker struct {
	clients        map[chan string]bool
	newClients     chan chan string
	defunctClients chan chan string
	messages       chan string
}

func (b *Broker) Start() {

	go func() {

		for {
			select {

			case s := <-b.newClients:
				b.clients[s] = true
				log.Println("Added new client")

			case s := <-b.defunctClients:
				delete(b.clients, s)
				log.Println("Removed client")

			case msg := <-b.messages:
				for s, _ := range b.clients {
					s <- msg
				}
				log.Printf("Broadcast message to %d clients", len(b.clients))
			}
		}
	}()
}

func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	messageChan := make(chan string)

	b.newClients <- messageChan

	notify := w.(http.CloseNotifier).CloseNotify()
	go func() {
		<-notify
		b.defunctClients <- messageChan
		log.Println("HTTP connection just closed.")
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {
		msg := <-messageChan
		fmt.Fprintf(w, "data: %s\n\n", msg)
		f.Flush()
	}

	log.Println("Finished HTTP request at ", r.URL.Path)
}

func MainPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatal("error parsing HTML template.")
	}

	// Render the template, writing to `w`.
	contents, _ := ioutil.ReadFile(filename)

	data := struct {
		Title    string
		ReadFile string
	}{
		Title:    filename,
		ReadFile: strings.TrimSpace(string(contents)),
	}

	t.Execute(w, data)

	log.Println("Finished HTTP request at ", r.URL.Path)
}

func main() {

	if len(os.Args) < 2 {
		fmt.Printf("Missing a file to tail")
		return
	}

	filename = os.Args[1]

	tail, err := tail.TailFile(filename, tail.Config{Follow: true, MustExist: true, Location: &tail.SeekInfo{0, os.SEEK_END}})

	if err != nil {
		log.Print(err.Error())
		return
	}

	b := &Broker{
		make(map[chan string]bool),
		make(chan (chan string)),
		make(chan (chan string)),
		make(chan string),
	}

	b.Start()
	http.Handle("/tails/", b)

	go func() {
		for line := range tail.Lines {
			b.messages <- fmt.Sprintf(line.Text)
			log.Printf("Sent message %d ", line.Text)
			// delay a little not to overwhelm the browser
			time.Sleep(0.5 * 1e9)

		}
	}()

	http.Handle("/", http.HandlerFunc(MainPageHandler))

	http.ListenAndServe(":8080", nil)
}
