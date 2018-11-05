package main

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/coreos/go-systemd/activation"
	"github.com/dcos/dcos-ui-update-service/config"
	our_http "github.com/dcos/dcos-ui-update-service/http"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

type UIService struct {
	Config *config.Config

	UIHandler *UIFileHandler

	UpdateManager *UpdateManager

	MasterCounter MasterCounter

	Client *our_http.Client
}

// SetupUIHandler create UIFileHandler for service ui and set default directory to
// the current downloaded version or the default document root
func SetupUIHandler(cfg *config.Config, um *UpdateManager) *UIFileHandler {
	documentRoot := cfg.DefaultDocRoot
	currentVersionPath, err := um.PathToCurrentVersion()
	if err == nil {
		documentRoot = currentVersionPath
	}
	return NewUIFileHandler(cfg.StaticAssetPrefix, documentRoot)
}

func setup(args []string) (*UIService, error) {
	cfg := config.Parse(args)
	httpClient, err := our_http.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not build http client: %s", err.Error())
		os.Exit(1)
	}
	updateManager, err := NewUpdateManager(cfg, httpClient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create update manager")
	}
	uiHandler := SetupUIHandler(cfg, updateManager)
	dcos := DCOS{
		MasterCountLocation: cfg.MasterCountFile,
	}

	return &UIService{
		Config:        cfg,
		UpdateManager: updateManager,
		UIHandler:     uiHandler,
		Client:        httpClient,
		MasterCounter: dcos,
	}, nil
}

// TODO: think about client timeouts https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
func main() {
	cliArgs := os.Args[1:]
	service, err := setup(cliArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initiate ui service, %s", err.Error())
		os.Exit(1)
	}

	// Use systemd socket activation.
	l, err := activation.Listeners()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to activate listeners from systemd, %s", err.Error())
		os.Exit(1)
	}

	if len(l) == 1 {
		// Run application
		if err := Run(service, l[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Application error: %s", err.Error())
			os.Exit(1)
		}
		return
	}
	fmt.Println("Did not receive any listeners from systemd, will start with configured listener instead.")
	// Start socket
	if err := StartSocket(service); err != nil {

		fmt.Fprintf(os.Stderr, "Application error: %s", err.Error())
		os.Exit(1)
	}
}

// Run serves the API based on the Config and Listener provided
func Run(service *UIService, l net.Listener) error {
	r := newRouter(service)
	http.Handle("/", r)
	loggedRouter := handlers.LoggingHandler(os.Stdout, r)
	return http.Serve(l, loggedRouter)
}

// StartSocket if systemd did not provide a socket
func StartSocket(service *UIService) error {
	listenNet := service.Config.ListenNetProtocol
	listenAddr := service.Config.ListenNetAddress

	l, err := net.Listen(listenNet, listenAddr)
	fmt.Fprintf(os.Stderr, "Starting new socket using net: %q and Addr: %q\n", listenNet, listenAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot listen for %q connections at address %q: %s \n", listenNet, listenAddr, err.Error())
		os.Exit(1)
	}
	return Run(service, l)
}

func newRouter(service *UIService) *mux.Router {
	assetPrefix := service.UIHandler.AssetPrefix()

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/", NotImplementedHandler)
	r.HandleFunc("/api/v1/update/{version}/", UpdateHandler(service))
	r.HandleFunc("/api/v1/reset/", ResetToDefaultUIHandler(service)).Methods("DELETE")
	r.PathPrefix(assetPrefix).Handler(service.UIHandler)

	return r
}

// NotImplementedHandler writes a HTTP Not Implemented response
func NotImplementedHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// UpdateHandler processes update requests
func UpdateHandler(service *UIService) func(http.ResponseWriter, *http.Request) {
	isMultiMaster, err := service.MasterCounter.IsMultiMaster()
	if err != nil {
		fmt.Printf("Error checking for multi master setup: %#v", err)
		return func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}

	// Unsupported multi master setup
	if isMultiMaster {
		return NotImplementedHandler
	}

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		version := vars["version"]

		// Check for empty version
		if len(version) == 0 {
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}
		service.Client.SetRequestHeaders(r.Header)
		defer service.Client.ClearRequestHeaders()

		err := service.UpdateManager.UpdateToVersion(version, service.UIHandler)

		if err != nil {
			// This returns locked on every error, it would be better if we would return a boolean if the process is locked
			w.WriteHeader(http.StatusLocked)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// ResetToDefaultUIHandler processes requests to reset to the default ui
func ResetToDefaultUIHandler(state *UIService) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// verify we aren't currently serving pre-bundled version
		if state.Config.DefaultDocRoot == state.UIHandler.DocumentRoot() {
			w.WriteHeader(http.StatusOK)
			return
		}
		err := state.UIHandler.UpdateDocumentRoot(state.Config.DefaultDocRoot)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = state.UpdateManager.ResetVersion()
		if err != nil {
			// TODO: Log we failed to remove latest
		}
		w.WriteHeader(http.StatusOK)
	}
}
