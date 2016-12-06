package authors

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/service-status-go/gtg"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

// AuthorHandler - struct for the handlers
type AuthorHandler struct {
	service AuthorService
}

// NewAuthorHandler - Create a new AuthorHandler
func NewAuthorHandler(service AuthorService) AuthorHandler {
	return AuthorHandler{service}
}

// GetAuthors - Return a JSON encoded list of all authors
func (h *AuthorHandler) GetAuthors(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Add("Content-Type", "application/json")
	if !h.service.isInitialised() {
		writeStatusServiceUnavailable(writer)
		return
	}

	if c, _ := h.service.getCount(); c == 0 {
		writeJSONMessageWithStatus(writer, "Authors not found", http.StatusNotFound)
		return
	}

	pv, err := h.service.getAuthors()

	if err != nil {
		writeJSONMessageWithStatus(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	defer pv.Close()
	writer.WriteHeader(http.StatusOK)
	io.Copy(writer, &pv)
}

// GetAuthorUUIDs - Get a list of JSON objects (not a JSON list) giving each id.
func (h *AuthorHandler) GetAuthorUUIDs(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Add("Content-Type", "application/json")
	if !h.service.isInitialised() {
		writeStatusServiceUnavailable(writer)
		return
	}

	if c, _ := h.service.getCount(); c == 0 {
		writeJSONMessageWithStatus(writer, "Authors not found", http.StatusNotFound)
		return
	}

	pv, err := h.service.getAuthorUUIDs()

	if err != nil {
		writeJSONMessageWithStatus(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	defer pv.Close()
	writer.WriteHeader(http.StatusOK)
	io.Copy(writer, &pv)
}

// GetCount - Get a count of the number of available authors
func (h *AuthorHandler) GetCount(writer http.ResponseWriter, req *http.Request) {
	if !h.service.isInitialised() {
		writer.Header().Add("Content-Type", "application/json")
		writeStatusServiceUnavailable(writer)
		return
	}
	count, err := h.service.getCount()
	if err != nil {
		writer.Header().Add("Content-Type", "application/json")
		writeJSONMessageWithStatus(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	writer.Write([]byte(strconv.Itoa(count)))
}

// HealthCheck - Return FT standard healthcheck
func (h *AuthorHandler) HealthCheck() v1a.Check {

	return v1a.Check{
		BusinessImpact:   "Unable to respond to requests",
		Name:             "Check service has finished initilising.",
		PanicGuide:       "https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/v1-authors-transformer",
		Severity:         1,
		TechnicalSummary: "Cannot serve any content as data not loaded.",
		Checker: func() (string, error) {
			if h.service.isInitialised() {
				return "Service is up and running", nil
			}
			return "Error as service initilising", errors.New("Service is initilising.")
		},
	}
}

// G2GCheck - Return FT standard good-to-go check
func (h *AuthorHandler) G2GCheck() gtg.Status {
	count, err := h.service.getCount()
	if h.service.isInitialised() && err == nil && count > 0 {
		return gtg.Status{GoodToGo: true}
	}
	return gtg.Status{GoodToGo: false}
}

// GetAuthorByUUID - Return the JSON for a single author
func (h *AuthorHandler) GetAuthorByUUID(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Add("Content-Type", "application/json")
	if !h.service.isInitialised() {
		writeStatusServiceUnavailable(writer)
		return
	}

	vars := mux.Vars(req)
	uuid := vars["uuid"]

	obj, found, err := h.service.getAuthorByUUID(uuid)
	if err != nil {
		writeJSONMessageWithStatus(writer, err.Error(), http.StatusInternalServerError)
	}
	writeJSONResponse(obj, found, writer)
}

// Reload - Reload the cache with fresh information
func (h *AuthorHandler) Reload(writer http.ResponseWriter, req *http.Request) {
	if !h.service.isInitialised() || !h.service.isDataLoaded() {
		writeStatusServiceUnavailable(writer)
		return
	}

	go func() {
		if err := h.service.reloadDB(); err != nil {
			log.Errorf("ERROR opening db: %v", err.Error())
		}
	}()
	writeJSONMessageWithStatus(writer, "Reloading authors", http.StatusAccepted)
}

func writeJSONResponse(obj interface{}, found bool, writer http.ResponseWriter) {
	if !found {
		writeJSONMessageWithStatus(writer, "Author not found", http.StatusNotFound)
		return
	}

	enc := json.NewEncoder(writer)
	if err := enc.Encode(obj); err != nil {
		log.Errorf("Error on json encoding=%v", err)
		writeJSONMessageWithStatus(writer, err.Error(), http.StatusInternalServerError)
		return
	}
}

func writeJSONMessageWithStatus(w http.ResponseWriter, msg string, statusCode int) {
	w.WriteHeader(statusCode)
	fmt.Fprintln(w, fmt.Sprintf("{\"message\": \"%s\"}", msg))
}

func writeStatusServiceUnavailable(w http.ResponseWriter) {
	writeJSONMessageWithStatus(w, "Service Unavailable", http.StatusServiceUnavailable)
}
