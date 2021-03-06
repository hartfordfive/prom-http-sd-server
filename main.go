package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/hartfordfive/prom-http-sd-server/config"
	"github.com/hartfordfive/prom-http-sd-server/handler"
	"github.com/hartfordfive/prom-http-sd-server/lib"
	"github.com/hartfordfive/prom-http-sd-server/logger"
	"github.com/hartfordfive/prom-http-sd-server/store"
	"github.com/hartfordfive/prom-http-sd-server/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var dataStore store.DataStore

var (
	metricHttpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "httpsdserver_req_duration_seconds",
		Help: "Duration of HTTP requests.",
	}, []string{"path"})
)

var (
	flagConfPath  *string
	flagDebug     *bool
	flagVersion   *bool
	log           *zap.Logger
	conf          *config.Config
	shutdownChan  chan bool
	interruptChan chan os.Signal
)

func init() {

	flagConfPath = flag.String(
		"conf",
		"/etc/prom-http-sd-server/prom-http-sd-server.conf",
		"When using local storage, path to storage dir.",
	)
	flagVersion = flag.Bool("version", false, "Show version and exit")
	flagDebug = flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()

	var log *zap.Logger
	var loggerErr error

	log, loggerErr = zap.NewProduction() // or NewExample, NewProduction, or NewDevelopment

	if *flagDebug {
		// If we're in debug mode, then create a dev logger instead
		log, loggerErr = zap.NewDevelopment() // or NewExample, NewProduction, or NewDevelopment
	}
	if loggerErr != nil {
		fmt.Printf("Could not initialize logger: %s\n", loggerErr)
		os.Exit(1)
	}
	logger.Logger = log
	defer logger.Logger.Sync()

	if *flagVersion {
		fmt.Printf("prom-http-sd-server %s (Git hash: %s)\n", version.Version, version.CommitHash)
		fmt.Printf("\tauthor: %s\n", version.Author)
		os.Exit(0)
	}

	if !lib.FileExists(*flagConfPath) {
		logger.Logger.Error(fmt.Sprintf("Error: Configuration '%s' not found\n", *flagConfPath))
		os.Exit(1)
	}

	interruptChan = make(chan os.Signal, 1)
	shutdownChan = make(chan bool, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)

	cnf, err := config.NewConfig(*flagConfPath)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("%s", err))
		os.Exit(1)
	}
	conf = cnf
}

// initDataStore creates a DataStore of any type supported by conf.StoreType
// Once, created the datastore is properly initialised.
// Finally, the store.StoreInstance gloabl variable is set to the newly created DataStore instance.
func initDataStore(storeType string) (err error) {
	switch storeType {
	case "local":
		store.StoreInstance, err = store.NewBoltDBDataStore(conf.LocalDBConfig.TargetStorePath, shutdownChan)

	case "consul":
		store.StoreInstance, err = store.NewConsulDataStore(conf.ConsulConfig.Host, conf.ConsulConfig.AllowStale, shutdownChan)

	default:
		err = fmt.Errorf("%s data store not implemented.", conf.StoreType)
	}

	return
}

// prometheusMiddleware implements mux.MiddlewareFunc.
func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()
		timer := prometheus.NewTimer(metricHttpDuration.WithLabelValues(path))
		next.ServeHTTP(w, r)
		timer.ObserveDuration()
	})
}

func main() {

	logger.Logger.Info("Starting prom-http-sd-server")

	// Should probably be changed too, we want to know about data store inits
	if conf.LocalDBConfig != nil {
		logger.Logger.Debug("Initializing data store",
			zap.String("config_path", conf.LocalDBConfig.TargetStorePath),
		)
	}

	config.GlobalConfig = conf

	// Init datastore
	logger.Logger.Sugar().Infof("Starting datastore (type: %s)...", conf.StoreType)
	err := initDataStore(conf.StoreType)
	if err != nil {
		logger.Logger.Error(err.Error())
		os.Exit(1)
	}

	// Init web server
	r := mux.NewRouter()
	r.HandleFunc("/api/target/{targetGroup}/{target}", handler.AddTargetHandler).Methods("POST")
	r.HandleFunc("/api/target/{targetGroup}/{target}", handler.RemoveTargetHandler).Methods("DELETE")
	r.HandleFunc("/api/target/{targetGroup}", handler.RemoveTargetGroupHandler).Methods("DELETE")
	r.HandleFunc("/api/labels/{targetGroup}", handler.GetTargetGroupLabelsHandler).Methods("GET")
	r.HandleFunc("/api/labels/update/{targetGroup}", handler.AddTargetGroupLabelsHandler).Methods("POST")
	r.HandleFunc("/api/labels/update/{targetGroup}/{label}", handler.RemoveTargetGroupLabelHandler).Methods("DELETE")
	r.HandleFunc("/api/targets", handler.ShowTargetsHandler).Methods("GET")
	r.HandleFunc("/debug_targets", handler.ShowDebugTargetsHandler).Methods("GET")
	r.HandleFunc("/debug_config", handler.ShowDebugConfigHandler).Methods("GET")
	r.Handle("/metrics", promhttp.Handler()).Methods("GET")
	r.HandleFunc("/health", handler.HealthHandler).Methods("GET")

	listenAddr := fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	logger.Logger.Info("prom-http-sd-server is now ready for connections",
		zap.String("address", listenAddr),
	)

	srv := &http.Server{
		Handler:      r,
		Addr:         listenAddr,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.Logger.Fatal(fmt.Sprintf("Error starting server: %v", err))
		}
	}()

	killSig := <-interruptChan
	switch killSig {
	case os.Interrupt, syscall.SIGTERM:
		close(shutdownChan)
	}

	dataStore.Shutdown()

	if err := srv.Shutdown(context.TODO()); err != nil {
		panic(err)
	}
	logger.Logger.Info("prom-http-sd-server shutdown complete")
}
