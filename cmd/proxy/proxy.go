package main

import "fmt"
import "flag" // has flag parsing
import "net/http"
import "log"
import "time"

// import "net/url"
import "net"
import "io"
import "context"

// https://eli.thegreenplace.net/2019/on-concurrency-in-go-http-servers
// golang accepts a new connection and delegates to a goroutine, this is how concurrency is achieved
type Server struct{}

func pipeConn(ch chan bool, clientConn *net.Conn, remoteConn *net.Conn) {
	bufSize := 72
	buf := make([]byte, bufSize)
  
  var dataProcessor = func (ch chan []byte, conn *net.Conn) {
    for {
      log.Println("waiting for data from: ", remoteConn)
      numBytesRead, err := io.ReadAtLeast(*conn, buf, bufSize)
      if numBytesRead > 0 {
        ch <- buf
      }

      if err == nil {
        continue
      }

      // we ran into an error
      switch err {
      case io.EOF:
        log.Println("done reading data from remote")
      default:
        log.Println(err)
      }
      break
    }
    ch <- nil
    close(ch)
  }

  sender := func (ch chan []byte, outputConn *net.Conn) {
    for {
      data := <- ch
      log.Println("sending: ", data)
      n, err := (*outputConn).Write(data)
      log.Println("sent: ", n)
      if err != nil {
        log.Println("unable to send data to ", outputConn)
        close(ch)
      }
    }
    log.Println("done sending for ", outputConn)
  }

  out := make(chan []byte)
  in := make(chan []byte)
  toClient := make(chan []byte)
  toRemote := make(chan []byte)

  go dataProcessor(in, clientConn)
  go dataProcessor(out, remoteConn)
  go sender(toClient, clientConn)
  go sender(toRemote, remoteConn)

	for {
  select {
  case fromClient, ok := <-in:
    log.Println("received from client")
    if !ok {
      log.Println("client closed?")
      ch <- true
    }

    log.Println("sending to remote")
    toRemote <- fromClient
  case fromRemote, ok := <-out:
    log.Println("received from remote")
    if !ok {
      log.Println("remote closed?")
      ch <- true
    }
    toClient <- fromRemote
  }
}

}


func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodConnect {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
	// well... the request path is parsed as a protocol relative path
	log.Println(r.URL.Host)

	// create the connection
  clientConn := GetConnFromRequest(r)
  log.Println(clientConn.RemoteAddr())

	remoteConn, err := net.Dial("tcp", r.URL.Host)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

  // https://stackoverflow.com/questions/48267616/why-is-golang-http-responsewriter-execution-being-delayed
  if flusher, ok := w.(http.Flusher); ok {
    log.Println("flushing http")
    flusher.Flush()
  }

  ch := make(chan bool)
  go pipeConn(ch, &clientConn, &remoteConn)

  var done bool;
  done, done = <-ch, <-ch
  
	fmt.Println("done", done)
}

type ClientConnKeyType string

var ClientConnContextKey = ClientConnKeyType("ClientConn")

func SaveClientConn(ctx context.Context, c net.Conn) context.Context {
  log.Println("save conn from", c)
	return context.WithValue(ctx, ClientConnContextKey, c)
}

// from https://stackoverflow.com/questions/29531993/accessing-the-underlying-socket-of-a-net-http-response
func GetConnFromRequest(r *http.Request) net.Conn {
	return r.Context().Value(ClientConnContextKey).(net.Conn)
}

func main() {
	// sets up socket at specified port
	// the way an http proxy works is by reciving a CONNECT request
	// it takes the connect request and opens up a socket to that domain
	// it then creates a tunnel between the client and the destination server
	// its main responsibilities are to send data from client to destination server
	// receive data from destination server and then send data to client
	// [client] <--> [proxy] <--> [dest]

	// read the parameters to this app
	var (
		proxyPort int
	)
	flag.IntVar(&proxyPort, "proxy-port", 7777, "the port which the proxy listens at")
	flag.Parse()

	fmt.Printf("%d\n", proxyPort)

	// setup listener at specified port
	// http.HandleFunc("/", http.StripPrefix("/", func (writer http.ResponseWriter, request *http.Request) {
	//   log.Printf("lol")
	//   fmt.Fprintf(writer, "%d\n", "lol")
	// }))

	handler := Server{}

	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", proxyPort),
		Handler:        handler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
		ConnContext:    SaveClientConn,
	}
	log.Fatal(s.ListenAndServe())

	// this doesn't work because we need CONNECT domain:port HTTP/1.1 to map to something
	// log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", proxyPort), nil))
	// it will be an HTTP listener, so we need to handle the MethodConnect

	// Now, does golangs http serve function serve concurrent requests?
	// How do we handle keeping state?
}
