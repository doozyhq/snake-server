package main

import (
	"flag"
	"net"
	"net/http"
	"runtime"

	"bitbucket.org/pushkin_ivan/pool-websocket-handler"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"golang.org/x/net/context"
)

func main() {
	flag.Parse()

	// Working listener is used for game servering
	workingListener, err :=
		net.Listen("tcp", Config.Host+":"+Config.Port)
	if err != nil {
		glog.Exitln("Cannot create working listener:", err)
	}

	// Shutdown listener is used only for shutdown command. Listening
	// only for local requests
	shutdownListener, err :=
		net.Listen("tcp", "127.0.0.1:"+Config.ShutdownPort)
	if err != nil {
		glog.Exitln("Cannot create shutdown listener:", err)
	}

	// Gets root context and cancel func for all goroutines on server
	cxt, cancel := context.WithCancel(context.Background())

	// Init pool factory
	factory, err := NewPGPoolFactory(cxt, Config.ConnLimit,
		Config.PgW, Config.PgH)
	if err != nil {
		glog.Exitln("Cannot create pool factory:", err)
	}

	// Init pool manager which allocates connections on pools
	poolManager, err := NewGamePoolManager(factory, Config.PoolLimit)
	if err != nil {
		glog.Exitln("Cannot create pool manager:", err)
	}

	streamer, err := NewStreamer(cxt, Config.Delay)
	if err != nil {
		glog.Exitln("Cannot create streamer:", err)
	}

	// Init connection manager
	connManager, err := NewConnManager(streamer)
	if err != nil {
		glog.Exitln("Cannot create connection manager:", err)
	}

	// Init request verifier
	verifier := NewVerifier(Config.HashSalt)

	// Configure websocket upgrader
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  Config.WsReadBufferSize,
		WriteBufferSize: Config.WsWriteBufferSize,
		// Don't check origin on tests
		CheckOrigin: func(*http.Request) bool { return true },
	}

	// Create pool handler
	handler := pwshandler.NewPoolHandler(
		poolManager, connManager, verifier, upgrader)

	// Setup GOMAXPROCS
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Start goroutine looking for shutdown command
	go func() {
		// Waiting for shutdown command. We don't need of connection
		if _, err := shutdownListener.Accept(); err != nil {
			glog.Errorln("Accepting shutdown connection:", err)
		}

		// Closing shutdown listener
		if err := shutdownListener.Close(); err != nil {
			glog.Errorln("Closing shutdown listener:", err)
		}

		// Finishing all goroutines
		cancel()

		// Closing working listener
		if err := workingListener.Close(); err != nil {
			glog.Errorln("Closing working listener:", err)
		}
	}()

	// Start server
	if err = http.Serve(workingListener, handler); err != nil {
		glog.Errorln("Servering error:", err)
	}
}
