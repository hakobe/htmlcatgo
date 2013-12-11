package main

import "net/http"
import "log"
import "os"
import "bufio"
import "fmt"
import "html/template"

type Client struct {
	out  chan string
	quit chan bool
}

type Broadcaster struct {
	addClient    chan (*Client)
	removeClient chan (*Client)
	sem          chan bool
}

func NewBroadcaster(stream *bufio.Reader) *Broadcaster {
	sem := make(chan bool, 1)
	sem <- true
	broadcaster := &Broadcaster{make(chan (*Client)), make(chan (*Client)), sem}

	out := make(chan string)
	quit := make(chan bool)
	go func() {
		for {
			line, err := stream.ReadString('\n')
			log.Print(line)
			if err != nil {
				log.Print(err)
				quit <- true
				return
			}
			out <- line
		}
	}()

	go func() {
		clients := [](* Client){}
		for {
			select {
			case line := <-out:
				for _, client := range clients {
					client.out <- line
				}
			case <-quit:
				<-broadcaster.sem
				for _, client := range clients {
					client.quit <- true
				}
				close(broadcaster.sem)
				return
			case c := <-broadcaster.addClient:
				clients = append(clients, c)
				log.Printf("Current # of clients: %d", len(clients))

				broadcaster.sem <- true
			case c := <-broadcaster.removeClient:
				newClients := make([](*Client), 0, len(clients))
				for _, ch := range clients {
					if ch != c {
						newClients = append(newClients, ch)
					}
				}
				clients = newClients
				log.Printf("Current # of clients: %d", len(clients))

				broadcaster.sem <- true
			}
		}
	}()

	return broadcaster
}

func (broadcaster *Broadcaster) AddClient(client *Client) bool {
	_, ok := <-broadcaster.sem
	if !ok {
		return false
	}

	broadcaster.addClient <- client

	return true
}

func (broadcaster *Broadcaster) RemoveClient(client *Client) bool {
	_, ok := <-broadcaster.sem
	if !ok {
		return false
	}

	broadcaster.removeClient <- client

	return true
}

func handleStream(res http.ResponseWriter, req *http.Request, broadcaster *Broadcaster) {
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

	client := &Client{out: make(chan string), quit: make(chan bool)}
	if !broadcaster.AddClient(client) {
		return
	}

	for {
		select {
		case line := <-client.out:
			fmt.Fprintf(res, "data:%s\n", line)
			fmt.Fprint(res, "\n")
			f.Flush()
		case <-client.quit:
			log.Println("Stream end")
			return
		case <-closer:
			broadcaster.RemoveClient(client)
			return
		}
	}
}

func main() {

	broadcaster := NewBroadcaster(bufio.NewReader(os.Stdin))

	http.HandleFunc("/stream", func(res http.ResponseWriter, req *http.Request) {
		log.Println("New client arrived")

		handleStream(res, req, broadcaster)

		log.Println("Client has been disconnected")
	})

	http.Handle("/js/", http.FileServer(http.Dir("./static/")))
	http.Handle("/css/", http.FileServer(http.Dir("./static/")))

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		var indexTemplate = template.Must(template.ParseFiles("templates/index.html"))
		indexTemplate.Execute(res, nil)
	})

	log.Println("starting server at http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
