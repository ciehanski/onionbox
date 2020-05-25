package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ciehanski/onionbox/onionbox"
	"github.com/ciehanski/onionbox/onionstore"
)

func main() {
	// Create onionbox instance that stores config
	ob := onionbox.Onionbox{
		Logger: log.New(os.Stdout, "[onionbox] ", log.LstdFlags|log.Lshortfile),
		Store:  onionstore.NewStore(),
	}

	// Init flags
	flag.BoolVar(&ob.Debug, "debug", false, "run in debug mode")
	flag.BoolVar(&ob.TorVersion3, "torv3", true, "use version 3 of the Tor circuit (recommended)")
	flag.IntVar(&ob.RemotePort, "rport", 80, "remote port used to host the onion service")
	flag.IntVar(&ob.LocalPort, "lport", 0, "local port used to host the onion service")
	flag.StringVar(&ob.TorrcFile, "torrc", "", "provide a custom torrc file for the onion service")
	flag.Parse()

	// Wait at most 3 minutes to publish the service
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Init Tor connection
	t, onionSvc, err := ob.Init(ctx)
	if err != nil {
		ob.Logf("Error starting Tor & initializing onion service: %v", err)
		ob.Quit()
	}
	defer func() {
		if err = onionSvc.Close(); err != nil {
			ob.Logf("Error closing connection to onion service: %v", err)
			ob.Quit()
		}
		if err = t.Close(); err != nil {
			ob.Logf("Error closing connection to Tor: %v", err)
			ob.Quit()
		}
	}()

	//Create a separate go routine which infinitely loops through the store to check for
	//expired buffer entries, and delete them.
	go func() {
		if err = ob.Store.DestroyExpiredBuffers(); err != nil {
			ob.Logf("Error destroying expired buffers: %v", err)
		}
	}()

	// Display the onion service URL
	ob.OnionURL = onionSvc.ID
	fmt.Printf("Please open a Tor capable browser and navigate to http://%v.onion\n", ob.OnionURL)

	srvErrCh := make(chan error, 1)
	go func() { srvErrCh <- ob.Server.Serve(onionSvc) }() // Begin serving
	if err = <-srvErrCh; err != nil {
		ob.Logf("Error serving on onion service: %v", err)
		ob.Quit()
	}
	defer func() { // Proper server shutdown when program ends
		if err = ob.Server.Shutdown(ctx); err != nil {
			ob.Logf("Error shutting down onionbox server: %v", err)
			ob.Quit()
		}
	}()
}
