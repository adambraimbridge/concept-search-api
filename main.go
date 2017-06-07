package main

import (
	"net/http"
	"os"
	"time"

	"github.com/Financial-Times/concept-search-api/resources"
	"github.com/Financial-Times/concept-search-api/service"
	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/http-handlers-go/httphandlers"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
	"github.com/rcrowley/go-metrics"
)

func main() {
	app := cli.App("concept-search-api", "API for searching concepts")
	port := app.String(cli.StringOpt{
		Name:   "port",
		Value:  "8080",
		Desc:   "Port to listen on",
		EnvVar: "PORT",
	})
	accessKey := app.String(cli.StringOpt{
		Name:   "aws-access-key",
		Desc:   "AWS ACCES KEY",
		EnvVar: "AWS_ACCESS_KEY_ID",
	})
	secretKey := app.String(cli.StringOpt{
		Name:   "aws-secret-access-key",
		Desc:   "AWS SECRET ACCES KEY",
		EnvVar: "AWS_SECRET_ACCESS_KEY",
	})
	esEndpoint := app.String(cli.StringOpt{
		Name:   "elasticsearch-endpoint",
		Value:  "http://localhost:9200",
		Desc:   "AES endpoint",
		EnvVar: "ELASTICSEARCH_ENDPOINT",
	})
	esAuth := app.String(cli.StringOpt{
		Name:   "auth",
		Value:  "none",
		Desc:   "Authentication method for ES cluster (aws or none)",
		EnvVar: "AUTH",
	})
	esIndex := app.String(cli.StringOpt{
		Name:   "elasticsearch-index",
		Value:  "concept",
		Desc:   "Elasticsearch index",
		EnvVar: "ELASTICSEARCH_INDEX",
	})
	searchResultLimit := app.Int(cli.IntOpt{
		Name:   "search-result-limit",
		Value:  50,
		Desc:   "The maximum number of search results returned",
		EnvVar: "RESULT_LIMIT",
	})

	log.SetLevel(log.InfoLevel)

	app.Action = func() {
		logStartupConfig(port, esEndpoint, esAuth, esIndex, searchResultLimit)

		search := service.NewEsConceptSearchService(*esIndex)
		conceptFinder := newConceptFinder(*esIndex, *searchResultLimit)
		healthcheck := newEsHealthService()

		if *esAuth == "aws" {
			go service.AWSClientSetup(*accessKey, *secretKey, *esEndpoint, time.Minute, search, conceptFinder, healthcheck)
		} else {
			go service.SimpleClientSetup(*esEndpoint, time.Minute, search, conceptFinder, healthcheck)
		}

		handler := resources.NewHandler(search)
		routeRequest(port, conceptFinder, handler, healthcheck)
	}

	log.SetLevel(log.InfoLevel)
	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("App could not start, error=[%s]\n", err)
		return
	}
}

func logStartupConfig(port, esEndpoint, esAuth, esIndex *string, searchResultLimit *int) {
	log.Info("Concept Search API uses the following configurations:")
	log.Infof("port: %v", *port)
	log.Infof("elasticsearch-endpoint: %v", *esEndpoint)
	log.Infof("elasticsearch-auth: %v", *esAuth)
	log.Infof("elasticsearch-index: %v", *esIndex)
	log.Infof("search-result-limit: %v", *searchResultLimit)
}

func routeRequest(port *string, conceptFinder conceptFinder, handler *resources.Handler, healthService *esHealthService) {
	servicesRouter := mux.NewRouter()
	servicesRouter.HandleFunc("/concept/search", conceptFinder.FindConcept).Methods("POST")
	servicesRouter.HandleFunc("/concepts", handler.ConceptSearch).Methods("GET")

	var monitoringRouter http.Handler = servicesRouter
	monitoringRouter = httphandlers.TransactionAwareRequestLoggingHandler(log.StandardLogger(), monitoringRouter)
	monitoringRouter = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, monitoringRouter)

	http.HandleFunc("/__health", v1a.Handler("Amazon Elasticsearch Service Healthcheck", "Checks for AES", healthService.connectivityHealthyCheck(), healthService.clusterIsHealthyCheck()))
	http.HandleFunc("/__health-details", healthService.healthDetails)
	http.HandleFunc("/__gtg", healthService.goodToGo)

	http.Handle("/", monitoringRouter)

	log.Infof("Concept Search API listening on port %v...", *port)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatalf("Unable to start: %v", err)
	}
}
