package main

import (
	"net/http"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

var (
	router = mux.NewRouter()
)

func (c *Context) startHealthcheck() {
	router.HandleFunc("/", c.healtcheck).Methods("GET", "HEAD").Name("Healthcheck")
	logrus.Info("Healthcheck handler is listening on ", healtcheckPort)
	logrus.Fatal(http.ListenAndServe(":"+strconv.Itoa(healtcheckPort), router))
}

func (c *Context) healtcheck(w http.ResponseWriter, req *http.Request) {

	_, err := c.Rancher.Client.GetSelfStack()
	if err != nil {
		logrus.Error("Healtcheck failed: unable to reach metadata")
		http.Error(w, "Failed to reach metadata server", http.StatusInternalServerError)
	} else {
		_, err := c.Consul.Ping()
		if err != nil {
			logrus.Errorf("Failed to reach Consul API: %v", err)
			http.Error(w, "Failed to reach Consul API ", http.StatusInternalServerError)
		} else {
			w.Write([]byte("OK"))
		}
	}
}
