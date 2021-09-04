package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/hartfordfive/prom-http-sd-server/config"
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
	metricTargetGroupUpdates = promauto.NewCounter(prometheus.CounterOpts{
		Name: "httpsdserver_target_group_updates",
		Help: "Number of times a target group has been updated.",
	})
	metricTargetGroupUpdatesFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "httpsdserver_target_group_updates_failed",
		Help: "Number of times a target group updated failed",
	})

	metricTargetRemove = promauto.NewCounter(prometheus.CounterOpts{
		Name: "httpsdserver_target_delete",
		Help: "Number of times a target was deleted",
	})
	metricTargetRemoveFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "httpsdserver_target_delete_failed",
		Help: "Number of times the deletion of a target failed",
	})

	metricTargetGroupLabelsUpdates = promauto.NewCounter(prometheus.CounterOpts{
		Name: "httpsdserver_target_group_labels_updates",
		Help: "Number of times the labels of a target group has been updated.",
	})
	metricTargetGroupLabelsUpdatesFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "httpsdserver_target_group_labels_updates_failed",
		Help: "Number of times an update to the labels of a target group has failed.",
	})
)

var (
	flagConfPath *string
	flagDebug    *bool
	flagVersion  *bool
	log          *zap.Logger
	conf         *config.Config
)

func init() {

	flagConfPath = flag.String(
		"conf-path",
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
		fmt.Errorf("Could not initialize logger: %s", loggerErr)
		os.Exit(1)
	}
	logger.Logger = log
	defer logger.Logger.Sync()

}

var HealthHandler = func(w http.ResponseWriter, req *http.Request) {
	/*
		TO COMPLETE:
		This handler should only return OK if the underlying datastore is ready to accept connections
	*/
	fmt.Fprint(w, "OK\n")
}

var AddTargetHandler = func(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	target := vars["target"]
	targetGroup := vars["targetGroup"]

	logger.Logger.Debug(fmt.Sprintf("Adding target %s to target list %s\n", target, targetGroup))
	if err := dataStore.AddTargetToGroup(targetGroup, target); err != nil {
		metricTargetGroupUpdatesFailed.Inc()
	} else {
		metricTargetGroupUpdates.Inc()
	}
	fmt.Fprintf(w, "OK\n")
}

var RemoveTargetHandler = func(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	target := vars["target"]
	targetGroup := vars["targetGroup"]

	logger.Logger.Debug(fmt.Sprintf("Adding target %s to target list %s\n", target, targetGroup))
	if err := dataStore.RemoveTargetFromGroup(targetGroup, target); err != nil {
		metricTargetRemoveFailed.Inc()
	} else {
		metricTargetRemove.Inc()
	}
	fmt.Fprintf(w, "OK\n")
}

var AddTargetGroupLabelsHandler = func(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetGroup := vars["targetGroup"]
	//labels := r.URL.Query().Get("lv_pairs")
	qsargs := r.URL.Query()

	dat := qsargs["labels"]
	labels := map[string]string{}
	for _, lvpair := range dat {
		parts := strings.Split(lvpair, "=")
		labels[parts[0]] = parts[1]
	}

	if err := dataStore.AddLabelsToGroup(targetGroup, labels); err != nil {
		metricTargetGroupLabelsUpdatesFailed.Inc()
	} else {
		metricTargetGroupLabelsUpdates.Inc()
	}
	fmt.Fprintf(w, "OK\n")
}

var RemoveTargetGroupLabelHandler = func(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetGroup := vars["targetGroup"]
	label := vars["label"]

	if err := dataStore.RemoveLabelFromGroup(targetGroup, label); err != nil {
		metricTargetGroupLabelsUpdatesFailed.Inc()
		http.Error(w, "ERROR", http.StatusInternalServerError)
		return
	}
	metricTargetGroupLabelsUpdates.Inc()
	fmt.Fprintf(w, "OK\n")
}

var ShowTargetsHandler = func(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	res, err := dataStore.Serialize(false)
	if err != nil {
		fmt.Fprint(w, "[]\n")
		return
	}
	fmt.Fprintf(w, "%s\n", res)
}

var ShowDebugTargetsHandler = func(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	res, err := dataStore.Serialize(true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	modifiedData := map[string]interface{}{}
	err = json.Unmarshal([]byte(res), &modifiedData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response, err := json.MarshalIndent(modifiedData, " ", " ")
	fmt.Fprintf(w, "%s\n", string(response))
}

var ShowDebugConfigHandler = func(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	// modifiedData := map[string]interface{}{}
	// modifiedData["config"] = conf
	// response, err := json.MarshalIndent(modifiedData, " ", " ")

	printCnf, err := conf.Serialize()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%s\n", printCnf)
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

	if *flagVersion {
		fmt.Printf("prom-http-sd-server %s (Git hash: %s)\n", version.Version, version.CommitHash)
		fmt.Printf("\tauthor: %s\n", version.Author)
		os.Exit(0)
	}

	if !lib.FileExists(*flagConfPath) {
		logger.Logger.Error(fmt.Sprintf("Error: Configuration '%s' not found\n", *flagConfPath))
		os.Exit(1)
	}

	logger.Logger.Info("Starting prom-http-sd-server")

	conf, err := config.NewConfig(*flagConfPath)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("%s", err))
		os.Exit(1)
	}

	if conf.LocalDBConfig != nil {
		logger.Logger.Debug("Initializing data store",
			zap.String("config_path", conf.LocalDBConfig.TargetStorePath),
		)
	}

	interruptChan := make(chan os.Signal, 1)
	shutdownChan := make(chan bool, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)

	//ar dsInitErr error = nil

	if conf.StoreType == "local" {
		dataStore = store.NewBoltDBDataStore(conf.LocalDBConfig.TargetStorePath, shutdownChan)
	} else if conf.StoreType == "consul" {
		fmt.Println(conf.ConsulConfig.Host)
		dataStore, err = store.NewConsulDataStore(conf.ConsulConfig.Host, conf.ConsulConfig.AllowStale, shutdownChan)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Could not use %s data store: %s", conf.StoreType, err.Error()))
			os.Exit(1)
		}
	} else {
		logger.Logger.Error(fmt.Sprintf("%s data store not implemented.", conf.StoreType))
		os.Exit(1)
	}
	defer dataStore.Shutdown()

	r := mux.NewRouter()

	r.HandleFunc("/api/target/{targetGroup}/{target}", AddTargetHandler).Methods("POST")
	r.HandleFunc("/api/target/{targetGroup}/{target}", RemoveTargetHandler).Methods("DELETE")
	r.HandleFunc("/api/labels/update/{targetGroup}", AddTargetGroupLabelsHandler).Methods("POST")
	r.HandleFunc("/api/labels/update/{targetGroup}/{label}", RemoveTargetGroupLabelHandler).Methods("DELETE")
	r.HandleFunc("/api/targets", ShowTargetsHandler).Methods("GET")
	r.HandleFunc("/debug_targets", ShowDebugTargetsHandler).Methods("GET")
	r.HandleFunc("/debug_config", ShowDebugConfigHandler).Methods("GET")
	r.Handle("/metrics", promhttp.Handler()).Methods("GET")
	r.HandleFunc("/health", HealthHandler).Methods("GET")

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

	if err := srv.Shutdown(context.TODO()); err != nil {
		panic(err)
	}
	logger.Logger.Info("prom-http-sd-server shutdown complete")

}
