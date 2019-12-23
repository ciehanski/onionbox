package onionbox

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/cretz/bine/tor"
	"github.com/ipsn/go-libtor"
	"github.com/natefinch/lumberjack"
	"golang.org/x/sys/unix"

	"github.com/ciehanski/onionbox/onionstore"
)

const (
	formCSRF   = "token"
	cookieCSRF = "X-CSRF-Token"
)

type Onionbox struct {
	OnionURL    string
	RemotePort  int
	TorVersion3 bool
	Store       *onionstore.OnionStore
	Logger      *log.Logger
	Server      *http.Server
	Debug       bool
}

func (ob *Onionbox) Init(ctx context.Context) (*tor.Tor, *tor.OnionService, error) {
	// If debug is NOT enabled, write all logs to disk (instead of stdout)
	// and rotate them when necessary.
	var torLogger io.Writer
	var lj io.Writer
	if !ob.Debug {
		if runtime.GOOS != "windows" {
			lj = &lumberjack.Logger{
				Filename:   "/var/log/onionbox/onionbox.log",
				MaxSize:    100, // megabytes
				MaxBackups: 3,
				MaxAge:     28, // days
				Compress:   true,
			}
			ob.Logger.SetOutput(lj)
			torLogger = lj
		} else {
			lj = &lumberjack.Logger{
				Filename:   "%APPDATA%/Local/onionbox/onionbox.log",
				MaxSize:    100, // megabytes
				MaxBackups: 3,
				MaxAge:     28, // days
				Compress:   true,
			}
			ob.Logger.SetOutput(lj)
			torLogger = lj
		}
	} else {
		ob.Logger.SetOutput(os.Stdout)
		torLogger = os.Stderr
	}

	// Start Tor
	t, err := startTor(torLogger)
	if err != nil {
		return nil, nil, err
	}

	// Start listening over onion service
	onionSvc, err := ob.listenTor(ctx, t)
	if err != nil {
		return nil, nil, err
	}

	// Init serving
	http.HandleFunc("/", ob.Router)
	ob.Server = &http.Server{
		// TODO: comeback. Tor is quite slow and depending on the size of the files being
		//  transferred, the server could timeout. I would like to keep set timeouts, but
		//  will need to find a sweet spot or enable an option for large transfers.
		IdleTimeout:  time.Minute * 3,
		ReadTimeout:  time.Minute * 3,
		WriteTimeout: time.Minute * 3,
		Handler:      nil,
	}

	return t, onionSvc, nil
}

func startTor(logger io.Writer) (*tor.Tor, error) {
	fmt.Println("Starting and registering onion service, please wait...")

	var tempDataDir string
	if runtime.GOOS != "windows" {
		tempDataDir = "/tmp"
	} else {
		tempDataDir = "%TEMP%"
	}

	t, err := tor.Start(nil, &tor.StartConf{ // Start tor
		ProcessCreator:         libtor.Creator,
		DebugWriter:            logger,
		UseEmbeddedControlConn: runtime.GOOS != "windows", // This option is not supported on Windows
		TempDataDirBase:        tempDataDir,
		RetainTempDataDir:      false,
	})
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (ob *Onionbox) listenTor(ctx context.Context, t *tor.Tor) (*tor.OnionService, error) {
	// Create an onion service to listen on any port but show as 80
	onionSvc, err := t.Listen(ctx, &tor.ListenConf{
		Version3:    ob.TorVersion3,
		RemotePorts: []int{ob.RemotePort},
	})
	if err != nil {
		return nil, err
	}
	return onionSvc, nil
}

// createCSRF creates a simple md5 hash which I use to avoid CSRF attacks when presenting HTML forms
func createCSRF() (string, error) {
	hasher := md5.New()
	_, err := io.WriteString(hasher, strconv.FormatInt(time.Now().Unix(), 10))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// Logf is a helper function which will utilize the Logger from ob
// to print formatted logs.
func (ob *Onionbox) Logf(format string, args ...interface{}) {
	ob.Logger.Printf(format, args...)
}

// Quit will Quit all stored buffers and exit onionbox.
func (ob *Onionbox) Quit() {
	if err := ob.Store.DestroyAll(); err != nil {
		ob.Logf("Error destroying all buffers from Store: %v", err)
	}
	os.Exit(0)
}

// DisableCoreDumps disables core dumps on Unix systems.
// ref: https://github.com/awnumar/memguard/blob/master/memcall/memcall_unix.go
func (ob *Onionbox) DisableCoreDumps() {
	if runtime.GOOS != "windows" {
		if err := unix.Setrlimit(unix.RLIMIT_CORE, &unix.Rlimit{Cur: 0, Max: 0}); err != nil {
			ob.Logf("Error disabling core dumps: %v", err)
		}
	}
}
