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

type Runner struct {
	clients [](* Client)
	sem chan bool
}

func RunCommand(name string, args ...string) (* Runner) {
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

	sem := make(chan bool, 1)
	sem <- true

	runner := &Runner{ [](* Client){}, sem }

	go func() {
		for {
			select {
			case line := <-command:
				for _,client := range runner.clients {
					client.command <-line
				}
			case <-quit:
				for _,client := range runner.clients {
					client.quit <-true
				}
				runner.Close()
				return;
			}
		}
	}();

	return runner
}

func (runner *Runner) AddClient(client *Client) bool {
	_,ok := <-runner.sem
	if (!ok) {
		return false
	}

	runner.clients = append(runner.clients, client )
	log.Printf("Current # of clients: %d", len(runner.clients))

	runner.sem <- true
	return true
}

func (runner *Runner) RemoveClient(client *Client) bool {
	_,ok := <-runner.sem
	if (!ok) {
		return false
	}

	newClients := make([](* Client), 0, len(runner.clients))
	for _,ch := range runner.clients {
		if (ch != client) {
			newClients = append(newClients, ch)
		}
	}
	runner.clients = newClients
	log.Printf("Current # of clients: %d", len(runner.clients))

	runner.sem <- true
	return true
}

func (runner *Runner) Close() {
	_,ok := <-runner.sem
	if (!ok) {
		return
	}
	close(runner.sem)
}

func handleStream(res http.ResponseWriter, req *http.Request, runner *Runner ) {
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

	client := &Client{ command : make(chan string), quit : make(chan bool) }
	if (!runner.AddClient(client)) {
		return
	}

	for {
		select {
		case line := <-client.command:
			fmt.Fprintf(res, "data:%s\n", line)
			fmt.Fprint(res, "\n")
			f.Flush()
		case <-client.quit:
			log.Println("command exit")
			return
		case <-closer:
			runner.RemoveClient(client)
			return
		}
	}
}

func main() {
	runner := RunCommand("tail", "-f", "/tmp/hoge")

	http.HandleFunc( "/stream", func(res http.ResponseWriter, req *http.Request) {
		log.Println("New client arrived")

		handleStream( res, req, runner )	

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
