package main

import (
	"flag"
	"fmt"
	"time"

	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"k8s.io/client-go/kubernetes"
	api "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig = flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")
	address    = flag.String("address", ":8000", "Address and port to bind the HTTP server to")
)

func main() {
	flag.Parse()

	inCluster := os.Getenv("INCLUSTER")

	var config *rest.Config
	if inCluster == "true" {
		log.Println("Using in-cluster configuration")
		icConfig, err := rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
		config = icConfig

	} else {
		log.Println("Using out-of-cluster configuration")
		oocConfig, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			panic(err.Error())
		}
		config = oocConfig
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	_, err = clientset.Deployments("default").List(api.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	log.Println("Config found, starting HTTP server")

	r := mux.NewRouter()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OK")
	})

	r.HandleFunc("/{namespace}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		deployments, err := clientset.Deployments(vars["namespace"]).List(api.ListOptions{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, deployment := range deployments.Items {
			fmt.Fprintf(w, "%s: %d, %d\n", deployment.Name, deployment.Status.Replicas, deployment.Spec.Replicas)
		}
	}).Methods("GET")

	r.HandleFunc("/{namespace}/{deployment}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		deployment, err := clientset.Deployments(vars["namespace"]).Get(vars["deployment"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		value, err := strconv.ParseInt(string(body), 10, 32)
		var size int32
		if err != nil {
			size = 0
		} else {
			size = int32(value)
		}

		deployment.Spec.Replicas = &size

		oldScale, err := clientset.Scales(vars["namespace"]).Get("Deployment", vars["deployment"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		oldScale.Spec = v1beta1.ScaleSpec{Replicas: size}
		newScale, err := clientset.Scales(vars["namespace"]).Update("Deployment", oldScale)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Scaled to %v\n", newScale.Spec.Replicas)

	}).Methods("PUT")

	loggedRouter := handlers.LoggingHandler(os.Stdout, r)
	srv := &http.Server{
		Handler: loggedRouter,
		Addr:    *address,
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
