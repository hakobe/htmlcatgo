package main

import "net/http"
import "log"

import "os/exec"

import "bufio"

import "fmt"
import "html/template"

type Client struct {
	command chan string
	quit chan bool
}

func NewClient() (* Client) {
	return &Client{ command : make(chan string), quit : make(chan bool) }
}

func main() {
	addClient, removeClient := runCommand("tail", "-f", "/tmp/hoge")

	http.HandleFunc( "/stream", func(res http.ResponseWriter, req *http.Request) {
		log.Println("New client arrived")

		client := NewClient()
		addClient <- client
		handleStream( res, req, client )	
		removeClient <- client 

		log.Println("Client has been disconnected")
	})

	http.Handle("/js/",  http.FileServer(http.Dir("./static/")))
	http.Handle("/css/", http.FileServer(http.Dir("./static/")))

	http.HandleFunc( "/", func(res http.ResponseWriter, req *http.Request) {
		var indexTemplate = template.Must( template.ParseFiles( "templates/index.html" ) )
		indexTemplate.Execute( res, nil )
	})

	log.Println("starting server at http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func runCommand(name string, args ...string) (addClient chan *Client, removeClient chan *Client) {
	cmd := exec.Command(name, args...)

	out, err := cmd.StdoutPipe()
	if (err != nil) {
		log.Fatal(err)
	}
	outBuf := bufio.NewReader(out)
	cmd.Start() 

	command := make(chan string)
	quit    := make(chan bool)
	go func() {
		for {
			line, err := outBuf.ReadString('\n')
			if (err != nil) {
				log.Print(err)
				log.Println("Command exited")
				cmd.Wait()
				quit <- true
				return
			}
			command <- line
		}
	}();

	addClient    = make(chan *Client)
	removeClient = make(chan *Client)
	go func() {
		clients := [](* Client){}
		for {
			select {
			case line := <-command:
				for _,client := range clients {
					client.command <-line
				}
			case <-quit:
				for _,client := range clients {
					client.quit <-true
				}
				clients = [](* Client){}
			case add := <-addClient:
				clients = append(clients, add )
				log.Printf("Current # of clients: %d", len(clients))
			case rem := <-removeClient:
				newClients := make([](* Client), 0, len(clients))
				for _,ch := range clients {
					if (ch != rem) {
						newClients = append(newClients, ch)
					}
				}
				clients = newClients
				log.Printf("Current # of clients: %d", len(clients))
			}
		}
	}();
	return addClient, removeClient
}

func handleStream(res http.ResponseWriter, req *http.Request, client *Client ) {
	f, ok := res.(http.Flusher)
	if !ok {
		http.Error(res, "Streaming unsupported", http.StatusInternalServerError)
		return
	}
	c, ok := res.(http.CloseNotifier)
	if !ok {
		http.Error(res, "Close notification unsupported", http.StatusInternalServerError)
		return
	}

	closer := c.CloseNotify()

	headers := res.Header()
	headers.Set("Content-Type", "text/event-stream; charset=utf-8")
	headers.Set("Cache-Control", "no-cache")

	for {
		select {
		case line := <-client.command:
			fmt.Fprintf(res, "data:%s\n", line)
			fmt.Fprint(res, "\n")
			f.Flush()
		case <-client.quit:
			log.Println("client quit")
			return
		case <-closer:
			return
		}
	}
}
