package main

import (
	"github.com/coreos/go-systemd/daemon"
	"github.com/itshosted/webutils/middleware"
	"github.com/itshosted/webutils/muxdoc"
	"github.com/itshosted/webutils/ratelimit"

	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type ScriptJob struct {
	File string
	Args []string
	Id   int64
}

type Status struct {
	Status string `json:"status"`
	Stdout string `json:"stdout"`
}

var (
	mux muxdoc.MuxDoc
	ln  net.Listener

	listen      string
	scriptQueue chan (ScriptJob)
)

func httpQueueAdd(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var (
		file string
		args string
	)

	if file = query.Get("file"); file == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		if _, e := w.Write([]byte(`{"error": "file-arg missing"}`)); e != nil {
			fmt.Printf("httpScript write e=%s\n", e.Error())
		}
		return
	}

	if args = query.Get("args"); args == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		if _, e := w.Write([]byte(`{"error": "args-arg missing"}`)); e != nil {
			fmt.Printf("httpScript write e=%s\n", e.Error())
		}
		return
	}

	if e := Validate("file", file); e != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		fmt.Printf("CRIT: httpScript got invalid file-pattern e=%s\n", e.Error())
		if _, e := w.Write([]byte(`{"error": "file invalid"}`)); e != nil {
			fmt.Printf("httpScript write e=%s\n", e.Error())
		}
		return
	}

	/* todo?? validating tickers here.. for _, arg := range strings.SplitN(args, ",", 100) {
		if e := Validate("ticker", arg); e != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(400)
			fmt.Printf("CRIT: httpScript got invalid arg-pattern e=%s\n", e.Error())
			if _, e := w.Write([]byte(`{"error": "arg invalid"}`)); e != nil {
				fmt.Printf("httpScript write e=%s\n", e.Error())
			}
			return
		}
	}*/

	if _, e := os.Stat(C.Scriptd + file); os.IsNotExist(e) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		fmt.Printf("CRIT: httpScript got file that does not exist e=%s\n", e.Error())
		if _, e := w.Write([]byte(`{"error": "no such file"}`)); e != nil {
			fmt.Printf("httpScript write e=%s\n", e.Error())
		}
		return
	}

	if cap(scriptQueue) == len(scriptQueue) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(503)
		fmt.Printf("CRIT: httpScript got a fully filled queue! cap=%d, len=%d\n", cap(scriptQueue), len(scriptQueue))
		if _, e := w.Write([]byte(`{"error": "queue full"}`)); e != nil {
			fmt.Printf("httpScript write e=%s\n", e.Error())
		}
		return
	}

	id, e := QueueAdd(args)
	if e != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		fmt.Printf("httpQueueAdd e=%s\n", e.Error())
		if _, e := w.Write([]byte(`{"error": "failed reserving position in DB"}`)); e != nil {
			fmt.Printf("httpScript write e=%s\n", e.Error())
		}
		return
	}

	scriptQueue <- ScriptJob{
		File: file,
		Args: strings.SplitN(args, ",", 100),
		Id:   id,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf(`{"id": %d}`, id)))
}

func httpQueueStatus(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var (
		id string
	)

	if id = query.Get("id"); id == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		if _, e := w.Write([]byte(`{"error": "id-arg missing"}`)); e != nil {
			fmt.Printf("httpQueueStatus write e=%s\n", e.Error())
		}
		return
	}

	state := ""
	status, stdout, e := QueueStatus(id)
	if e == ErrNoRows {
		state = "NOTFOUND"
		e = nil
	}
	if e != nil {
		fmt.Printf("QueueStatus: %s\n", e.Error())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		if _, e := w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, e.Error()))); e != nil {
			fmt.Printf("httpQueueStatus: %s\n", e.Error())
		}
		return
	}

	if state == "" {
		if status == -1 {
			state = "PENDING"
		} else if status == 0 {
			state = "DONE"
		} else {
			state = "ERROR"
		}
	}

	s := Status{
		Status: state,
		Stdout: stdout,
	}
	enc := json.NewEncoder(w)

	w.Header().Set("Content-Type", "application/json")
	if e := enc.Encode(s); e != nil {
		fmt.Printf("httpQueueStatus: %s\n", e.Error())
	}
}

// Return API Documentation (paths)
func doc(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(404)
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(mux.String()))
}

func main() {
	cfgPath := ""
	flag.BoolVar(&Verbose, "v", false, "Show all that happens")
	flag.StringVar(&cfgPath, "c", "/etc/queued/config.toml", "Config-file")
	flag.Parse()

	if e := Init(cfgPath); e != nil {
		panic(e)
	}
	if Verbose {
		fmt.Printf("C=%+v\n", C)
	}

	scriptQueue = make(chan ScriptJob, C.QueueSize)

	mux.Title = "Queued-API"
	mux.Desc = "Async processing with privilege separation."
	mux.Add("/", doc, "This documentation")
	mux.Add("/queue/add", httpQueueAdd, "Run script, ?file=FILENAME&args=CLI-ARGS")
	mux.Add("/queue/status", httpQueueStatus, "Get status by ?id=UUID_FROM_ADD")

	middleware.Add(ratelimit.Use(5, 5))
	http.Handle("/", middleware.Use(mux.Mux))

	// Small worker to run shell execs
	go func() {
		for {
			req := <-scriptQueue
			// TODO: Move to config?
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			if Verbose {
				fmt.Printf("script.d req=%+v\n", req)
			}

			cmd := exec.CommandContext(ctx, C.Scriptd+req.File, req.Args...)
			out, e := cmd.CombinedOutput()
			if e != nil {
				fmt.Printf("cmd.Run() failed with e=%s, stdout/stderr:\n%s\n", e.Error(), out)
			}

			status := 0
			if e != nil {
				status = 1
			}
			if e := QueueUpdate(ctx, req.Id, status, out); e != nil {
				fmt.Printf("cmd.QueueUpdate e=%s\n", e.Error())
			}
		}
	}()

	var e error
	server := &http.Server{Addr: C.Listen, Handler: nil}
	ln, e = net.Listen("tcp", server.Addr)
	if e != nil {
		panic(e)
	}
	if Verbose {
		fmt.Println("httpd listening on " + C.Listen)
	}

	sent, e := daemon.SdNotify(false, "READY=1")
	if e != nil {
		panic(e)
	}
	if !sent {
		fmt.Printf("SystemD notify NOT sent\n")
	}

	if e := server.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)}); e != nil {
		panic(e)
	}
}
