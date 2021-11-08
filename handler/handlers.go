package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/hartfordfive/prom-http-sd-server/config"
	"github.com/hartfordfive/prom-http-sd-server/logger"
	"github.com/hartfordfive/prom-http-sd-server/store"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
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

var HealthHandler = func(w http.ResponseWriter, req *http.Request) {
	/*
		TO COMPLETE:
		This handler should only return OK if the underlying datastore is ready to accept connections
	*/
	fmt.Fprint(w, "OK")
}

var AddTargetHandler = func(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	target := vars["target"]
	targetGroup := vars["targetGroup"]

	dataStore := *store.StoreInstance

	logger.Logger.Debug(fmt.Sprintf("Adding target %s to target list %s\n", target, targetGroup))
	if err := dataStore.AddTargetToGroup(targetGroup, target); err != nil {
		metricTargetGroupUpdatesFailed.Inc()
	} else {
		metricTargetGroupUpdates.Inc()
	}
	fmt.Fprintf(w, "OK")
}

var RemoveTargetHandler = func(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	target := vars["target"]
	targetGroup := vars["targetGroup"]

	logger.Logger.Debug(fmt.Sprintf("Adding target %s to target list %s\n", target, targetGroup))
	dataStore := *store.StoreInstance
	if err := dataStore.RemoveTargetFromGroup(targetGroup, target); err != nil {
		metricTargetRemoveFailed.Inc()
	} else {
		metricTargetRemove.Inc()
	}
	fmt.Fprintf(w, "OK")
}

var RemoveTargetGroupHandler = func(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetGroup := vars["targetGroup"]

	logger.Logger.Debug(fmt.Sprintf("Removing target group %s\n", targetGroup))
	dataStore := *store.StoreInstance
	if err := dataStore.RemoveTargetGroup(targetGroup); err != nil {
		metricTargetRemoveFailed.Inc()
	} else {
		metricTargetRemove.Inc()
	}
	fmt.Fprintf(w, "OK")
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

	dataStore := *store.StoreInstance
	if err := dataStore.AddLabelsToGroup(targetGroup, labels); err != nil {
		metricTargetGroupLabelsUpdatesFailed.Inc()
	} else {
		metricTargetGroupLabelsUpdates.Inc()
	}
	fmt.Fprintf(w, "OK")
}

var GetTargetGroupLabelsHandler = func(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetGroup := vars["targetGroup"]

	dataStore := *store.StoreInstance
	dat, err := dataStore.GetTargetGroupLabels(targetGroup)

	if err != nil {
		fmt.Fprint(w, "[]\n")
		return
	}

	b, _ := json.MarshalIndent(dat, "", "    ")
	res := string(b)
	fmt.Fprintf(w, "%s\n", res)
}

var RemoveTargetGroupLabelHandler = func(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetGroup := vars["targetGroup"]
	label := vars["label"]

	dataStore := *store.StoreInstance
	if err := dataStore.RemoveLabelFromGroup(targetGroup, label); err != nil {
		metricTargetGroupLabelsUpdatesFailed.Inc()
		http.Error(w, "ERROR", http.StatusInternalServerError)
		return
	}
	metricTargetGroupLabelsUpdates.Inc()
	fmt.Fprintf(w, "OK")
}

var ShowTargetsHandler = func(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	dataStore := *store.StoreInstance
	res, err := dataStore.Serialize(false)
	if err != nil {
		fmt.Fprint(w, "[]\n")
		return
	}
	fmt.Fprintf(w, "%s\n", res)
}

var ShowDebugTargetsHandler = func(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	dataStore := *store.StoreInstance
	res, err := dataStore.Serialize(true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println(res)

	modifiedData := map[string]interface{}{}
	err = json.Unmarshal([]byte(res), &modifiedData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println(modifiedData)

	response, err := json.MarshalIndent(modifiedData, " ", " ")
	fmt.Fprintf(w, "%s\n", string(response))
}

var ShowDebugConfigHandler = func(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	// modifiedData := map[string]interface{}{}
	// modifiedData["config"] = conf
	// response, err := json.MarshalIndent(modifiedData, " ", " ")

	printCnf, err := config.GlobalConfig.Serialize()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%s\n", printCnf)
}
