package authors

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/service-status-go/gtg"
	log "github.com/sirupsen/logrus"
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
func (h *AuthorHandler) HealthCheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Unable to respond to requests",
		Name:             "Check service has finished initialising.",
		PanicGuide:       "TBD",
		Severity:         1,
		TechnicalSummary: "Cannot serve any content as data not loaded.",
		Checker: func() (string, error) {
			if h.service.isInitialised() {
				return "Service is up and running", nil
			}
			return "Error as service initialising", errors.New("Service is initialising.")
		},
	}
}

// GTG - Return FT standard good-to-go check
func (h *AuthorHandler) GTG() gtg.Status {
	statusCheck := func() gtg.Status {
		return gtgCheck(h.initChecker)
	}
	return gtg.FailFastParallelCheck([]gtg.StatusChecker{statusCheck})()
}

func gtgCheck(handler func() (string, error)) gtg.Status {
	if _, err := handler(); err != nil {
		return gtg.Status{GoodToGo: false, Message: err.Error()}
	}
	return gtg.Status{GoodToGo: true}
}

func (h *AuthorHandler) initChecker() (string, error) {
	count, err := h.service.getCount()
	if h.service.isInitialised() && err == nil && count > 0 {
		return "Service is initialised", err
	}
	return "Service is not initialised yet, or there was an error with the initialization", err
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
