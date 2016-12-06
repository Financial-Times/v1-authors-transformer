package authors

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/service-status-go/gtg"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

const (
	testUUID          = "bba39990-c78d-3629-ae83-808c333c6dbc"
	testUUID2         = "be2e7e2b-0fa2-3969-a69b-74c46e754032"
	getAuthorResponse = `{"uuid":"bba39990-c78d-3629-ae83-808c333c6dbc","prefLabel":"","type":"","alternativeIdentifiers":{}}
{"uuid":"be2e7e2b-0fa2-3969-a69b-74c46e754032","prefLabel":"","type":"","alternativeIdentifiers":{}}
`
	getAuthorUUIDsResponse = `{"ID":"bba39990-c78d-3629-ae83-808c333c6dbc"}
{"ID":"be2e7e2b-0fa2-3969-a69b-74c46e754032"}
`
	getAuthorByUUIDResponse = "{\"uuid\":\"bba39990-c78d-3629-ae83-808c333c6dbc\",\"prefLabel\":\"European Union\",\"type\":\"Organisation\",\"alternativeIdentifiers\":{\"TME\":[\"MTE3-U3ViamVjdHM=\"],\"uuids\":[\"bba39990-c78d-3629-ae83-808c333c6dbc\"]}}\n"
)

func TestHandlers(t *testing.T) {
	var wg sync.WaitGroup
	tests := []struct {
		name         string
		req          *http.Request
		dummyService AuthorService
		statusCode   int
		contentType  string // Contents of the Content-Type header
		body         string
	}{
		{"Success - get author by uuid",
			newRequest("GET", fmt.Sprintf("/transformers/authors/%s", testUUID)),
			&dummyService{
				found:       true,
				initialised: true,
				authors:     []author{{UUID: testUUID, PrefLabel: "European Union", AlternativeIdentifiers: alternativeIdentifiers{UUIDs: []string{testUUID}, TME: []string{"MTE3-U3ViamVjdHM="}}, Type: "Organisation"}}},
			http.StatusOK,
			"application/json",
			getAuthorByUUIDResponse},
		{"Not found - get author by uuid",
			newRequest("GET", fmt.Sprintf("/transformers/authors/%s", testUUID)),
			&dummyService{
				found:       false,
				initialised: true,
				authors:     []author{{}}},
			http.StatusNotFound,
			"application/json",
			"{\"message\": \"Author not found\"}\n"},
		{"Service unavailable - get author by uuid",
			newRequest("GET", fmt.Sprintf("/transformers/authors/%s", testUUID)),
			&dummyService{
				found:       false,
				initialised: false,
				authors:     []author{}},
			http.StatusServiceUnavailable,
			"application/json",
			"{\"message\": \"Service Unavailable\"}\n"},
		{"Success - get authors count",
			newRequest("GET", "/transformers/authors/__count"),
			&dummyService{
				found:       true,
				count:       1,
				initialised: true,
				authors:     []author{{UUID: testUUID}}},
			http.StatusOK,
			"application/json",
			"1"},
		{"Failure - get authors count",
			newRequest("GET", "/transformers/authors/__count"),
			&dummyService{
				err:         errors.New("Something broke"),
				found:       true,
				count:       1,
				initialised: true,
				authors:     []author{{UUID: testUUID}}},
			http.StatusInternalServerError,
			"application/json",
			"{\"message\": \"Something broke\"}\n"},
		{"Failure - get authors count not init",
			newRequest("GET", "/transformers/authors/__count"),
			&dummyService{
				err:         errors.New("Something broke"),
				found:       true,
				count:       1,
				initialised: false,
				authors:     []author{{UUID: testUUID}}},
			http.StatusServiceUnavailable,
			"application/json", "{\"message\": \"Service Unavailable\"}\n"},
		{"get authors - success",
			newRequest("GET", "/transformers/authors"),
			&dummyService{
				found:       true,
				initialised: true,
				count:       2,
				authors:     []author{{UUID: testUUID}, {UUID: testUUID2}}},
			http.StatusOK,
			"application/json",
			getAuthorResponse},
		{"get authors - Not found",
			newRequest("GET", "/transformers/authors"),
			&dummyService{
				initialised: true,
				count:       0,
				authors:     []author{}},
			http.StatusNotFound,
			"application/json",
			"{\"message\": \"Authors not found\"}\n"},
		{"get authors - Service unavailable",
			newRequest("GET", "/transformers/authors"),
			&dummyService{
				found:       false,
				initialised: false,
				authors:     []author{}},
			http.StatusServiceUnavailable,
			"application/json",
			"{\"message\": \"Service Unavailable\"}\n"},
		{"get authors IDS - Success",
			newRequest("GET", "/transformers/authors/__id"),
			&dummyService{
				found:       true,
				initialised: true,
				count:       1,
				authors:     []author{{UUID: testUUID}, {UUID: testUUID2}}},
			http.StatusOK,
			"application/json",
			getAuthorUUIDsResponse},
		{"get authors IDS - Not found",
			newRequest("GET", "/transformers/authors/__id"),
			&dummyService{
				initialised: true,
				count:       0,
				authors:     []author{}},
			http.StatusNotFound,
			"application/json",
			"{\"message\": \"Authors not found\"}\n"},
		{"get authors IDS - Service unavailable",
			newRequest("GET", "/transformers/authors/__id"),
			&dummyService{
				found:       false,
				initialised: false,
				authors:     []author{}},
			http.StatusServiceUnavailable,
			"application/json",
			"{\"message\": \"Service Unavailable\"}\n"},
		{"GTG unavailable - get GTG",
			newRequest("GET", status.GTGPath),
			&dummyService{
				found:       false,
				initialised: false,
				authors:     []author{}},
			http.StatusServiceUnavailable,
			"application/json",
			""},
		{"GTG unavailable - get GTG but no authors",
			newRequest("GET", status.GTGPath),
			&dummyService{
				found:       false,
				initialised: true},
			http.StatusServiceUnavailable,
			"application/json",
			""},
		{"GTG unavailable - get GTG count returns error",
			newRequest("GET", status.GTGPath),
			&dummyService{
				found:       false,
				initialised: true,
				err:         errors.New("Count error")},
			http.StatusServiceUnavailable,
			"application/json",
			""},
		{"GTG OK - get GTG",
			newRequest("GET", status.GTGPath),
			&dummyService{
				found:       true,
				initialised: true,
				count:       2},
			http.StatusOK,
			"application/json",
			"OK"},
		{"Health bad - get Health check",
			newRequest("GET", "/__health"),
			&dummyService{
				found:       false,
				initialised: false},
			http.StatusOK,
			"application/json",
			"regex=Service is initilising"},
		{"Health good - get Health check",
			newRequest("GET", "/__health"),
			&dummyService{
				found:       false,
				initialised: true},
			http.StatusOK,
			"application/json",
			"regex=Service is up and running"},
		{"Reload accepted - request reload",
			newRequest("POST", "/transformers/authors/__reload"),
			&dummyService{
				wg:          &wg,
				initialised: true,
				dataLoaded:  true},
			http.StatusAccepted,
			"application/json",
			"{\"message\": \"Reloading authors\"}\n"},
		{"Reload accepted even though error loading data in background.",
			newRequest("POST", "/transformers/authors/__reload"),
			&dummyService{
				wg:          &wg,
				err:         errors.New("Boom goes the backend..."),
				initialised: true,
				dataLoaded:  true},
			http.StatusAccepted,
			"application/json",
			"{\"message\": \"Reloading authors\"}\n"},
		{"Reload - Service unavailable as not initialised",
			newRequest("POST", "/transformers/authors/__reload"),
			&dummyService{
				wg:          &wg,
				err:         errors.New("Boom goes the backend..."),
				initialised: false,
				dataLoaded:  true},
			http.StatusServiceUnavailable,
			"application/json",
			"{\"message\": \"Service Unavailable\"}\n"},
		{"Reload - Service unavailable as data not loaded",
			newRequest("POST", "/transformers/authors/__reload"),
			&dummyService{
				wg:          &wg,
				err:         errors.New("Boom goes the backend..."),
				initialised: true,
				dataLoaded:  false},
			http.StatusServiceUnavailable,
			"application/json",
			"{\"message\": \"Service Unavailable\"}\n"},
	}
	for _, test := range tests {
		wg.Add(1)
		rec := httptest.NewRecorder()
		router(test.dummyService).ServeHTTP(rec, test.req)
		assert.Equal(t, test.statusCode, rec.Code, fmt.Sprintf("%s: Wrong response code, was %d, should be %d", test.name, rec.Code, test.statusCode))

		b, err := ioutil.ReadAll(rec.Body)
		assert.NoError(t, err)
		body := string(b)
		if strings.HasPrefix(test.body, "regex=") {
			regex := strings.TrimPrefix(test.body, "regex=")
			matched, err := regexp.MatchString(regex, body)
			assert.NoError(t, err)
			assert.True(t, matched, fmt.Sprintf("Could not match regex:\n %s \nin body:\n %s", regex, body))
		} else {
			assert.Equal(t, test.body, body, fmt.Sprintf("%s: Wrong body", test.name))
		}
	}
}

func TestReloadIsCalled(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	rec := httptest.NewRecorder()
	s := &dummyService{
		wg:          &wg,
		found:       true,
		initialised: true,
		dataLoaded:  true,
		count:       2,
		authors:     []author{}}
	log.Infof("s.loadDBCalled: %v", s.loadDBCalled)
	router(s).ServeHTTP(rec, newRequest("POST", "/transformers/authors/__reload"))
	wg.Wait()
	assert.True(t, s.loadDBCalled)
}

func newRequest(method, url string) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	return req
}

type dummyService struct {
	found        bool
	authors      []author
	initialised  bool
	dataLoaded   bool
	count        int
	err          error
	loadDBCalled bool
	wg           *sync.WaitGroup
}

func (s *dummyService) loadCuratedAuthors(bAuthors []berthaAuthor) error {
	return nil
}

func (s *dummyService) getAuthors() (io.PipeReader, error) {
	pv, pw := io.Pipe()
	go func() {
		encoder := json.NewEncoder(pw)
		for _, sub := range s.authors {
			encoder.Encode(sub)
		}
		pw.Close()
	}()
	return *pv, nil
}

func (s *dummyService) getAuthorUUIDs() (io.PipeReader, error) {
	pv, pw := io.Pipe()
	go func() {
		encoder := json.NewEncoder(pw)
		for _, sub := range s.authors {
			encoder.Encode(authorUUID{UUID: sub.UUID})
		}
		pw.Close()
	}()
	return *pv, nil
}

func (s *dummyService) getAuthorLinks() (io.PipeReader, error) {
	pv, pw := io.Pipe()
	go func() {
		var links []authorLink
		for _, sub := range s.authors {
			links = append(links, authorLink{APIURL: "http://localhost:8080/transformers/authors/" + sub.UUID})
		}
		b, _ := json.Marshal(links)
		log.Infof("Writing bytes... %v", string(b))
		pw.Write(b)
		pw.Close()
	}()
	return *pv, nil
}

func (s *dummyService) getCount() (int, error) {
	return s.count, s.err
}

func (s *dummyService) getAuthorByUUID(uuid string) (author, bool, error) {
	return s.authors[0], s.found, nil
}

func (s *dummyService) isInitialised() bool {
	return s.initialised
}

func (s *dummyService) isDataLoaded() bool {
	return s.dataLoaded
}

func (s *dummyService) Shutdown() error {
	return s.err
}

func (s *dummyService) reloadDB() error {
	defer s.wg.Done()
	s.loadDBCalled = true
	return s.err
}

func router(s AuthorService) *mux.Router {
	m := mux.NewRouter()
	h := NewAuthorHandler(s)
	m.HandleFunc("/transformers/authors", h.GetAuthors).Methods("GET")
	m.HandleFunc("/transformers/authors/__count", h.GetCount).Methods("GET")
	m.HandleFunc("/transformers/authors/__reload", h.Reload).Methods("POST")
	m.HandleFunc("/transformers/authors/__id", h.GetAuthorUUIDs).Methods("GET")
	m.HandleFunc("/transformers/authors/{uuid}", h.GetAuthorByUUID).Methods("GET")
	m.HandleFunc("/__health", v1a.Handler("V1 Authors Transformer Healthchecks", "Checks for the health of the service", h.HealthCheck()))
	g2gHandler := status.NewGoodToGoHandler(gtg.StatusChecker(h.G2GCheck))
	m.HandleFunc(status.GTGPath, g2gHandler)
	return m
}
