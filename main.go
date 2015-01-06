// Copyright 2015 Pushkin Ivan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"math"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/ivan1993spb/pwshandler"
	"golang.org/x/net/context"
)

// Infolog leveles
const (
	INFOLOG_LEVEL_SERVER = iota + 1 // Server level
	INFOLOG_LEVEL_POOLS             // Pool level
	INFOLOG_LEVEL_CONNS             // Connection level
)

// Paths
const (
	// Path to game websocket
	PATH_TO_GAME = "/game.ws"

	// Server settings:

	PATH_TO_SERVER_LIMITS   = "/server_limits.json"
	PATH_TO_PLAYGROUND_SIZE = "/playground_size.json"

	// Working information:

	// Count of opened pools
	PATH_TO_POOL_COUNT = "/pool_count.json"
	// Count of opened connections on server
	PATH_TO_CONN_COUNT = "/conn_count.json"

	// List of pool ids with counts of opened connections on pool
	PATH_TO_POOL_INFO_LIST = "/pool_info_list.json"
	// Ids of opened connections in pool
	PATH_TO_POOL_CONN_IDS = "/pool_conn_ids.json"
)

type errStartingServer struct {
	err error
}

func (e *errStartingServer) Error() string {
	return "starting server error: " + e.err.Error()
}

func main() {

	/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
	 *                  BEGIN PARSING PARAMETERS                   *
	 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

	// Networking
	var host, mainPort, sdPort string

	flag.StringVar(&host, "host", "", "server host")
	flag.StringVar(&mainPort, "main_port", "8081",
		"port on which server handles external requests")
	flag.StringVar(&sdPort, "shutdown_port", "8082",
		"port on which server accepts shutdown request")

	// Security
	var (
		hashSalt        string
		checkUniqueConn bool
	)

	flag.StringVar(&hashSalt, "hash_salt", "",
		"hash salt for request verifying")
	flag.BoolVar(&checkUniqueConn, "check_unique_conn", false,
		"true to enable verifying connection uniqueness")

	// Server limits and playground size
	var poolLimit, connLimit, pgW, pgH uint

	flag.UintVar(&poolLimit, "pool_limit", 10,
		"max pool count on server")
	flag.UintVar(&connLimit, "conn_limit", 4,
		"max connection count on pool")
	flag.UintVar(&pgW, "pg_w", 40, "playground width")
	flag.UintVar(&pgH, "pg_h", 28, "playground height")

	// Handlers
	var handleServerLimits, handlePlaygroundSize, handlePoolCount,
		handleConnCount, handlePoolInfoList, handlePoolConnIds bool

	flag.BoolVar(&handleServerLimits, "handle_server_limits", false,
		"true to enable access to server limits")
	flag.BoolVar(&handlePlaygroundSize, "handle_pg_size", false,
		"true to enable access to playground size")
	flag.BoolVar(&handlePoolCount, "handle_pool_count", false,
		"true to enable access to pool count")
	flag.BoolVar(&handleConnCount, "handle_conn_count", false,
		"true to enable access to connection count")
	flag.BoolVar(&handlePoolInfoList, "handle_pool_info_list", false,
		"true to enable access to pool list")
	flag.BoolVar(&handlePoolConnIds, "handle_pool_conn_ids", false,
		"true to enable access to connection ids on selected pool")

	flag.Parse()

	if glog.V(INFOLOG_LEVEL_SERVER) {
		glog.Infoln("checking parameters")

		if len(host) == 0 {
			glog.Warningln("empty host")
		}
		if len(mainPort) == 0 {
			glog.Warningln("empty main port")
		} else if i, e := strconv.Atoi(mainPort); e != nil || i < 1 {
			glog.Warningln("invalid main port")
		}
		if len(sdPort) == 0 {
			glog.Warningln("empty shutdown port")
		} else if i, e := strconv.Atoi(sdPort); e != nil || i < 1 {
			glog.Warningln("invalid shutdown port")
		}

		if len(hashSalt) == 0 {
			glog.Warningln("empty hash salt; protection is disabled")
		}

		if poolLimit == 0 || poolLimit > math.MaxUint16 {
			glog.Warningln("invalid pool limit")
		}
		if connLimit == 0 || connLimit > math.MaxUint16 {
			glog.Warningln("invalid connection limit per pool")
		}
		if pgW*pgH == 0 {
			glog.Warningln("invalid playground size")
		}
		if pgW > math.MaxUint8 {
			glog.Warningln("playground width must be <=",
				math.MaxUint8)
		}
		if pgH > math.MaxUint8 {
			glog.Warningln("playground height must be <=",
				math.MaxUint8)
		}
	}

	/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
	 *                   END PARSING PARAMETERS                    *
	 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

	if glog.V(INFOLOG_LEVEL_SERVER) {
		glog.Infoln("preparing to start server")
	}

	/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
	 *                  BEGIN CREATING LISTENERS                   *
	 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

	mainListener, err := net.Listen("tcp", host+":"+mainPort)
	if err != nil {
		glog.Exitln(&errStartingServer{
			fmt.Errorf("cannot create main listener: %s", err),
		})
	}

	// Shutdown listener is used for shutdown command
	shutdownListener, err := net.Listen("tcp", "127.0.0.1:"+sdPort)
	if err != nil {
		glog.Exitln(&errStartingServer{
			fmt.Errorf("cannot create shutdown listener: %s", err),
		})
	}

	if glog.V(INFOLOG_LEVEL_SERVER) {
		glog.Infoln("listeners was created")
	}

	/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
	 *                   END CREATING LISTENERS                    *
	 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

	// Root context
	cxt, cancel := context.WithCancel(context.Background())

	/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
	 *                  BEGIN INIT GAME MODULES                    *
	 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

	// Init pool factory
	factory, err := NewPGPoolFactory(cxt, uint16(connLimit),
		uint8(pgW), uint8(pgH))
	if err != nil {
		glog.Exitln(&errStartingServer{err})
	}
	if glog.V(INFOLOG_LEVEL_SERVER) {
		glog.Infoln("pool factory was created")
	}

	// Init pool manager which allocates connections on pools
	poolManager, err := NewGamePoolManager(factory, uint16(poolLimit))
	if err != nil {
		glog.Exitln(&errStartingServer{err})
	}
	if glog.V(INFOLOG_LEVEL_SERVER) {
		glog.Infoln("pool manager was created")
	}

	// Init connection manager
	connManager := NewConnManager()
	if glog.V(INFOLOG_LEVEL_SERVER) {
		glog.Infoln("connection manager was created")
	}

	/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
	 *                   END INIT GAME MODULES                     *
	 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

	/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
	 *                    BEGIN INIT HANDLERS                      *
	 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

	var mux Mux = http.NewServeMux()

	// If hash salt is empty protection will disabled
	if len(hashSalt) > 0 {
		// Try to init security mux with token verifying
		if mux, err = NewSecurityMux(hashSalt); err != nil {
			glog.Exitln(&errStartingServer{err})
		}
		if glog.V(INFOLOG_LEVEL_SERVER) {
			glog.Infoln("security handler mux was created")
		}
	}

	if glog.V(INFOLOG_LEVEL_CONNS) {
		mux = &ReportMux{mux}
		glog.Infoln("report mux was created")
	}

	if glog.V(INFOLOG_LEVEL_SERVER) {
		glog.Infoln("root mux was created")
	}

	// Game handler is main and always is available
	if checkUniqueConn {
		mux.Handle(
			PATH_TO_GAME,
			UniqueRequestsHandler(
				pwshandler.PoolHandler(poolManager, connManager, nil),
				poolManager,
			),
		)
		if glog.V(INFOLOG_LEVEL_SERVER) {
			glog.Infoln("unique request handler was created")
		}
	} else {
		mux.Handle(
			PATH_TO_GAME,
			pwshandler.PoolHandler(poolManager, connManager, nil),
		)
	}
	if glog.V(INFOLOG_LEVEL_SERVER) {
		glog.Infoln("game handler was created")
	}

	// Server setting information handlers
	if handleServerLimits {
		mux.Handle(
			PATH_TO_SERVER_LIMITS,
			ServerLimitsHandler(poolLimit, connLimit),
		)
		if glog.V(INFOLOG_LEVEL_SERVER) {
			glog.Infoln("server limits handler was created")
		}
	}
	if handlePlaygroundSize {
		mux.Handle(
			PATH_TO_PLAYGROUND_SIZE,
			PlaygroundSizeHandler(uint8(pgW), uint8(pgH)),
		)
		if glog.V(INFOLOG_LEVEL_SERVER) {
			glog.Infoln("playgound size handler was created")
		}
	}

	// Working information handlers
	if handlePoolCount {
		mux.Handle(
			PATH_TO_POOL_COUNT,
			PoolCountHandler(poolManager),
		)
		if glog.V(INFOLOG_LEVEL_SERVER) {
			glog.Infoln("pool count handler was created")
		}
	}
	if handleConnCount {
		mux.Handle(
			PATH_TO_CONN_COUNT,
			ConnCountHandler(poolManager),
		)
		if glog.V(INFOLOG_LEVEL_SERVER) {
			glog.Infoln("connection count handler was created")
		}
	}
	if handlePoolInfoList {
		mux.Handle(
			PATH_TO_POOL_INFO_LIST,
			PoolInfoListHandler(poolManager),
		)
		if glog.V(INFOLOG_LEVEL_SERVER) {
			glog.Infoln("pool info list handler was created")
		}
	}
	if handlePoolConnIds {
		mux.Handle(
			PATH_TO_POOL_CONN_IDS,
			PoolConnIdsHandler(poolManager),
		)
		if glog.V(INFOLOG_LEVEL_SERVER) {
			glog.Infoln("pool connection ids handler was created")
		}
	}

	/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
	 *                     END INIT HANDLERS                       *
	 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

	runtime.GOMAXPROCS(runtime.NumCPU())

	if glog.V(INFOLOG_LEVEL_SERVER) {
		glog.Infoln("starting server")
	}

	// Start goroutine looking for shutdown command
	go func() {
		// Waiting for shutdown command
		if _, err := shutdownListener.Accept(); err != nil {
			glog.Errorln("accepting shutdown connection error:", err)
		}
		if glog.V(INFOLOG_LEVEL_SERVER) {
			glog.Infoln("accepted shutdown connection")
		}

		// Closing shutdown listener
		if err := shutdownListener.Close(); err != nil {
			glog.Errorln("closing shutdown listener error:", err)
		}
		if glog.V(INFOLOG_LEVEL_SERVER) {
			glog.Infoln("shutdown listener was closed")
		}

		// Finishing all goroutines
		if glog.V(INFOLOG_LEVEL_SERVER) {
			glog.Infoln("finishing all goroutines on server")
		}
		cancel()
		time.Sleep(time.Second)

		if glog.V(INFOLOG_LEVEL_SERVER) {
			glog.Infoln(
				"closing main listener;",
				"server will shutdown with error:",
				"use of closed network connection",
			)
		}
		// Closing main listener
		if err := mainListener.Close(); err != nil {
			glog.Errorln("closing main listener error:", err)
		}
	}()

	// Starting server
	err = http.Serve(mainListener, mux)
	if err != nil {
		glog.Errorln("servering error:", err)
	}

	// Flush log
	glog.Flush()

	time.Sleep(time.Second)

	if glog.V(INFOLOG_LEVEL_SERVER) {
		glog.Infoln("goodbye")
	}
}
