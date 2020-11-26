package main

import (
	"context"
	_ "errors"
	"flag"
	"fmt"
	"io"
	_ "io/ioutil"
	"log"
	"net/http"
	_ "net/url"
	"os"
	"os/signal"
	"strconv"
	_ "sync"
	"sync/atomic"
	"syscall"
	"time"
)

// уникальный идентификатор для горутин вебсервера
var request_counter uint64

// то, что будет отдаваться как заголовок server name
const server_name string = "HTTP-srv v0.1"

var h_server *http.Server
var h_server_created int
var log_file_opened int
var free_resurses_done int32

//
//
// хендлеры
//----------------------------------------------------------------------
// дефолтный хендлер
//
func h_default(w http.ResponseWriter, r *http.Request) {

	request_id := atomic.AddUint64(&request_counter, 1)

	w.Header().Set("rid", strconv.FormatUint(request_id, 10))
	w.Header().Set("Server", server_name)
	fmt.Fprintf(w, "-\n\n\n")
}

//----------------------------------------------------------------------
//
//

func free_resurses() {

	if atomic.CompareAndSwapInt32(&free_resurses_done, 0, 1) {

		log.Printf("free_resurses() begin\n")

		if h_server_created == 1 {

			ctx_1, ctx_1_cancel_fnc := context.WithTimeout(context.Background(), 40*time.Second)
			defer ctx_1_cancel_fnc()

			log.Printf("server Shutdown\n")
			ShutdownErr := h_server.Shutdown(ctx_1)
			if ShutdownErr != nil {

				log.Printf("server Shutdown Failed:%+v\n", ShutdownErr)
				log.Printf("server Close\n")
				h_server.Close()

			} else {

				log.Printf("server Shutdown comlite\n")
			}
		}

		log.Printf("free_resurses() end\n")

		atomic.CompareAndSwapInt32(&free_resurses_done, 1, 2)
	}
}

var (
	fl_http_port    int
	fl_http_portPtr *int
	fl_log_file     string
	LOG_FILE        *os.File
)

func main() {

	free_resurses_done = 0
	h_server_created = 0
	request_counter = 0
	log_file_opened = 0
	pid := os.Getpid()

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	fl_http_portPtr = flag.Int("http_port", 0, "http server port ( 1 - 65534 )")
	flag.StringVar(&fl_log_file, "log_file", "", "path to log file")
	flag.Parse()

	if (*fl_http_portPtr < 1) || (*fl_http_portPtr > 65534) {

		log.Printf("bad http port!!! exit now!!!\n")
		os.Exit(100)
	}
	fl_http_port = *fl_http_portPtr

	if fl_log_file != "" {

		var fopenErr error
		LOG_FILE, fopenErr = os.OpenFile(fl_log_file, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		if fopenErr != nil {

			log.Printf("failed OpenFile %s:%+v\n", fl_log_file, fopenErr)

		} else {

			log_file_opened = 1
			log.SetOutput(io.Writer(LOG_FILE))
		}
	}

	// вешаем обработку сигналов
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		s := <-sigChan
		// поймали один из сигналов, освобождаем реурсы
		log.Printf("-- %s signal --\n", s.String())
		free_resurses()
	}()

	log.Printf("pid %d\n", pid)
	log.Printf("app started\n")

	h_server = &http.Server{
		Addr:           ":" + fmt.Sprint(fl_http_port),
		ReadTimeout:    35 * time.Second,
		WriteTimeout:   35 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	h_server_created = 1

	// привязываем хендлеры
	http.HandleFunc("/", h_default)

	// запускаем веб-сервер
	var listen_err error
	listen_err = h_server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	log.Printf("%+v\n", listen_err.Error())

	// освобождаем ресурсы,
	// если они не были освобождены
	// при обработке сигнала
	free_resurses()

	// дожидаемся окончания освобождения ресурсов
	// если они до сих пор освобождаются в
	// обработчике сигнала
	for {
		tst := atomic.LoadInt32(&free_resurses_done)
		if tst > 1 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("app ended\n\n\n")
	if log_file_opened == 1 {

		log.SetOutput(io.Writer(os.Stdout))
		LOG_FILE.Close()
	}
}
