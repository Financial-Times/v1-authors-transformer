package main

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/http-handlers-go/httphandlers"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/Financial-Times/tme-reader/tmereader"
	"github.com/Financial-Times/v1-authors-transformer/authors"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
	"github.com/rcrowley/go-metrics"
	"github.com/sethgrid/pester"
	log "github.com/sirupsen/logrus"
)

func main() {
	app := cli.App("v1-authors-transformer", "A RESTful API for transforming TME Authors to UP json")
	username := app.String(cli.StringOpt{
		Name:   "tme-username",
		Value:  "",
		Desc:   "TME username used for http basic authentication",
		EnvVar: "TME_USERNAME",
	})
	password := app.String(cli.StringOpt{
		Name:   "tme-password",
		Value:  "",
		Desc:   "TME password used for http basic authentication",
		EnvVar: "TME_PASSWORD",
	})
	token := app.String(cli.StringOpt{
		Name:   "token",
		Value:  "",
		Desc:   "Token to be used for accessig TME",
		EnvVar: "TOKEN",
	})
	baseURL := app.String(cli.StringOpt{
		Name:   "base-url",
		Value:  "http://localhost:8080/transformers/authors/",
		Desc:   "Base url",
		EnvVar: "BASE_URL",
	})
	tmeBaseURL := app.String(cli.StringOpt{
		Name:   "tme-base-url",
		Value:  "https://tme.ft.com",
		Desc:   "TME base url",
		EnvVar: "TME_BASE_URL",
	})
	port := app.Int(cli.IntOpt{
		Name:   "port",
		Value:  8080,
		Desc:   "Port to listen on",
		EnvVar: "PORT",
	})
	maxRecords := app.Int(cli.IntOpt{
		Name:   "maxRecords",
		Value:  int(10000),
		Desc:   "Maximum records to be queried to TME",
		EnvVar: "MAX_RECORDS",
	})
	batchSize := app.Int(cli.IntOpt{
		Name:   "batchSize",
		Value:  int(10),
		Desc:   "Number of requests to be executed in parallel to TME",
		EnvVar: "BATCH_SIZE",
	})
	cacheFileName := app.String(cli.StringOpt{
		Name:   "cache-file-name",
		Value:  "cache.db",
		Desc:   "Cache file name",
		EnvVar: "CACHE_FILE_NAME",
	})
	graphiteTCPAddress := app.String(cli.StringOpt{
		Name:   "graphiteTCPAddress",
		Value:  "",
		Desc:   "Graphite TCP address, e.g. graphite.ft.com:2003. Leave as default if you do NOT want to output to graphite (e.g. if running locally)",
		EnvVar: "GRAPHITE_ADDRESS",
	})
	graphitePrefix := app.String(cli.StringOpt{
		Name:   "graphitePrefix",
		Value:  "",
		Desc:   "Prefix to use. Should start with content, include the environment, and the host name. e.g. content.test.public.content.by.concept.api.ftaps59382-law1a-eu-t",
		EnvVar: "GRAPHITE_PREFIX",
	})
	logMetrics := app.Bool(cli.BoolOpt{
		Name:   "logMetrics",
		Value:  false,
		Desc:   "Whether to log metrics. Set to true if running locally and you want metrics output",
		EnvVar: "LOG_METRICS",
	})
	berthaSrcURL := app.String(cli.StringOpt{
		Name:   "bertha-source-url",
		Desc:   "The URL of the Bertha Authors JSON source",
		EnvVar: "BERTHA_SOURCE_URL",
	})

	tmeTaxonomyName := "Authors"

	app.Action = func() {
		baseftrwapp.OutputMetricsIfRequired(*graphiteTCPAddress, *graphitePrefix, *logMetrics)
		client := getResilientClient()
		modelTransformer := new(authors.AuthorTransformer)
		s := authors.NewAuthorService(
			tmereader.NewTmeRepository(
				client,
				*tmeBaseURL,
				*username,
				*password,
				*token,
				*maxRecords,
				*batchSize,
				tmeTaxonomyName,
				&tmereader.AuthorityFiles{},
				modelTransformer),
			*baseURL,
			tmeTaxonomyName,
			*maxRecords,
			*cacheFileName,
			*berthaSrcURL,
			client)
		defer s.Shutdown()
		handler := authors.NewAuthorHandler(s)
		router(handler)

		log.Printf("listening on %d", *port)
		err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
		if err != nil {
			log.Errorf("Error by listen and serve: %v", err.Error())
		}
	}
	app.Run(os.Args)
}

func router(handler authors.AuthorHandler) {
	servicesRouter := mux.NewRouter()

	servicesRouter.HandleFunc("/transformers/authors", handler.GetAuthors).Methods("GET")
	servicesRouter.HandleFunc("/transformers/authors/__count", handler.GetCount).Methods("GET")
	servicesRouter.HandleFunc("/transformers/authors/__ids", handler.GetAuthorUUIDs).Methods("GET")
	servicesRouter.HandleFunc("/transformers/authors/__reload", handler.Reload).Methods("POST")
	servicesRouter.HandleFunc("/transformers/authors/{uuid:([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})}", handler.GetAuthorByUUID).Methods("GET")

	var monitoringRouter http.Handler = servicesRouter
	monitoringRouter = httphandlers.TransactionAwareRequestLoggingHandler(log.StandardLogger(), monitoringRouter)
	monitoringRouter = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, monitoringRouter)
	timedHC := fthealth.TimedHealthCheck{
		HealthCheck: fthealth.HealthCheck{
			SystemCode:  "v1-authors-tf",
			Name:        "V1 Authors Transformer",
			Description: "It pulls and transforms V1/TME Authors into the UPP JSON model of an Author.",
			Checks:      []fthealth.Check{handler.HealthCheck()},
		},
		Timeout: 10 * time.Second,
	}

	http.HandleFunc("/__health", fthealth.Handler(timedHC))
	http.HandleFunc(status.PingPath, status.PingHandler)
	http.HandleFunc(status.PingPathDW, status.PingHandler)
	http.HandleFunc(status.BuildInfoPath, status.BuildInfoHandler)
	http.HandleFunc(status.BuildInfoPathDW, status.BuildInfoHandler)
	http.HandleFunc(status.GTGPath, status.NewGoodToGoHandler(handler.GTG))
	http.Handle("/", monitoringRouter)
}

func getResilientClient() *pester.Client {
	tr := &http.Transport{
		MaxIdleConnsPerHost: 32,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
	}
	c := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}
	client := pester.NewExtendedClient(c)
	client.Backoff = pester.ExponentialBackoff
	client.MaxRetries = 5
	client.Concurrency = 1

	return client
}
