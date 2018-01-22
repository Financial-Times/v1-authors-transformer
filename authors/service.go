package authors

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Financial-Times/tme-reader/tmereader"
	log "github.com/sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/jaytaylor/html2text"
	"github.com/pborman/uuid"
)

const (
	cacheBucket = "author"
)

type httpClient interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

// AuthorService - interface for retrieving v1 authors
type AuthorService interface {
	getAuthors() (io.PipeReader, error)
	getAuthorLinks() (io.PipeReader, error)
	getAuthorUUIDs() (io.PipeReader, error)
	getAuthorByUUID(uuid string) (author, bool, error)
	getCount() (int, error)
	isInitialised() bool
	isDataLoaded() bool
	reloadDB() error
	Shutdown() error
	loadCuratedAuthors([]berthaAuthor) error
}

type authorServiceImpl struct {
	sync.RWMutex
	repository    tmereader.Repository
	baseURL       string
	taxonomyName  string
	maxTmeRecords int
	initialised   bool
	dataLoaded    bool
	cacheFileName string
	db            *bolt.DB
	berthaURL     string
	httpClient    httpClient
}

// NewAuthorService - create a new AuthorService
func NewAuthorService(repo tmereader.Repository, baseURL string, taxonomyName string, maxTmeRecords int, cacheFileName string, berthaURL string, httpClient httpClient) AuthorService {
	s := &authorServiceImpl{
		repository:    repo,
		baseURL:       baseURL,
		taxonomyName:  taxonomyName,
		maxTmeRecords: maxTmeRecords,
		initialised:   true,
		cacheFileName: cacheFileName,
		berthaURL:     berthaURL,
		httpClient:    httpClient}
	s.setDataLoaded(false)
	go func(service *authorServiceImpl) { service.reloadDB() }(s)
	return s
}

func (s *authorServiceImpl) isInitialised() bool {
	s.RLock()
	defer s.RUnlock()
	return s.initialised
}

func (s *authorServiceImpl) setInitialised(val bool) {
	s.Lock()
	s.initialised = val
	s.Unlock()
}

func (s *authorServiceImpl) isDataLoaded() bool {
	s.RLock()
	defer s.RUnlock()
	return s.dataLoaded
}

func (s *authorServiceImpl) setDataLoaded(val bool) {
	s.Lock()
	s.dataLoaded = val
	s.Unlock()
}

func (s *authorServiceImpl) Shutdown() error {
	log.Info("Shuting down...")
	s.Lock()
	defer s.Unlock()
	s.initialised = false
	s.dataLoaded = false
	if s.db == nil {
		return errors.New("DB not open")
	}
	return s.db.Close()
}

func (s *authorServiceImpl) getCount() (int, error) {
	s.RLock()
	defer s.RUnlock()
	if !s.isDataLoaded() {
		return 0, nil
	}

	var count int
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(cacheBucket))
		if bucket == nil {
			return fmt.Errorf("Bucket %v not found!", cacheBucket)
		}
		count = bucket.Stats().KeyN
		return nil
	})
	return count, err
}

func (s *authorServiceImpl) getAuthors() (io.PipeReader, error) {
	s.RLock()
	pv, pw := io.Pipe()
	go func() {
		defer s.RUnlock()
		defer pw.Close()
		s.db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(cacheBucket))
			c := b.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				if _, err := pw.Write(v); err != nil {
					return err
				}
				io.WriteString(pw, "\n")
			}
			return nil
		})
	}()
	return *pv, nil
}

func (s *authorServiceImpl) getAuthorUUIDs() (io.PipeReader, error) {
	s.RLock()
	pv, pw := io.Pipe()
	go func() {
		defer s.RUnlock()
		defer pw.Close()
		s.db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(cacheBucket))
			c := b.Cursor()
			encoder := json.NewEncoder(pw)
			for k, _ := c.First(); k != nil; k, _ = c.Next() {
				if k == nil {
					break
				}
				pl := authorUUID{UUID: string(k[:])}
				if err := encoder.Encode(pl); err != nil {
					return err
				}
			}
			return nil
		})
	}()
	return *pv, nil
}

func (s *authorServiceImpl) getAuthorLinks() (io.PipeReader, error) {
	s.RLock()
	pv, pw := io.Pipe()
	go func() {
		defer s.RUnlock()
		defer pw.Close()
		io.WriteString(pw, "[")
		s.db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(cacheBucket))
			c := b.Cursor()
			encoder := json.NewEncoder(pw)
			var k []byte
			k, _ = c.First()
			for {
				if k == nil {
					break
				}
				pl := authorLink{APIURL: s.baseURL + "/" + string(k[:])}
				if err := encoder.Encode(pl); err != nil {
					return err
				}
				if k, _ = c.Next(); k != nil {
					io.WriteString(pw, ",")
				}
			}
			return nil
		})
		io.WriteString(pw, "]")
	}()
	return *pv, nil
}

func (s *authorServiceImpl) getAuthorByUUID(uuid string) (author, bool, error) {
	s.RLock()
	defer s.RUnlock()
	var cachedValue []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(cacheBucket))
		if bucket == nil {
			return fmt.Errorf("Bucket %v not found!", cacheBucket)
		}
		cachedValue = bucket.Get([]byte(uuid))
		return nil
	})

	if err != nil {
		log.Errorf("ERROR reading from cache file for [%v]: %v", uuid, err.Error())
		return author{}, false, err
	}
	if len(cachedValue) == 0 {
		log.Infof("INFO No cached value for [%v].", uuid)
		return author{}, false, nil
	}

	var cachedAuthor author
	if err := json.Unmarshal(cachedValue, &cachedAuthor); err != nil {
		log.Errorf("ERROR unmarshalling cached value for [%v]: %v.", uuid, err.Error())
		return author{}, true, err
	}
	return cachedAuthor, true, nil
}

func (s *authorServiceImpl) openDB() error {
	s.Lock()
	defer s.Unlock()
	log.Infof("Opening database '%v'.", s.cacheFileName)
	if s.db == nil {
		var err error
		if s.db, err = bolt.Open(s.cacheFileName, 0600, &bolt.Options{Timeout: 1 * time.Second}); err != nil {
			log.Errorf("ERROR opening cache file for init: %v.", err.Error())
			return err
		}
	}
	return s.createCacheBucket()
}

func (s *authorServiceImpl) reloadDB() error {
	s.setDataLoaded(false)
	err := s.loadDB()
	if err != nil {
		log.Errorf("Error while creating AuthorService: [%v]", err.Error())
		return err
	}
	var bAuthors []berthaAuthor

	bAuthors, err = s.getBerthaAuthors(s.berthaURL)
	if err != nil {
		log.Errorf("Error on Bertha load: [%v]", err.Error())
		return err
	} else {
		err = s.loadCuratedAuthors(bAuthors)
		if err != nil {
			log.Errorf("Error while loading in the curated authors: [%v]", err.Error())
		}
	}

	return err
}

func (s *authorServiceImpl) loadDB() error {
	var wg sync.WaitGroup
	log.Info("Loading DB...")
	c := make(chan []author)
	go s.processAuthors(c, &wg)
	defer func(w *sync.WaitGroup) {
		close(c)
		w.Wait()
	}(&wg)

	if err := s.openDB(); err != nil {
		s.setInitialised(false)
		return err
	}

	responseCount := 0
	for {
		terms, err := s.repository.GetTmeTermsFromIndex(responseCount)
		if err != nil {
			return err
		}
		if len(terms) < 1 {
			log.Info("Finished fetching authors from TME. Waiting subroutines to terminate.")
			break
		}

		wg.Add(1)
		s.processTerms(terms, c)
		responseCount += s.maxTmeRecords
	}
	return nil
}

func (s *authorServiceImpl) processTerms(terms []interface{}, c chan<- []author) {
	log.Info("Processing terms...")
	var cacheToBeWritten []author
	for _, iTerm := range terms {
		t := iTerm.(term)
		cacheToBeWritten = append(cacheToBeWritten, transformAuthor(t, s.taxonomyName))
	}
	c <- cacheToBeWritten
}

func (s *authorServiceImpl) processAuthors(c <-chan []author, wg *sync.WaitGroup) {
	for authors := range c {
		log.Infof("Processing batch of %v authors.", len(authors))
		if err := s.db.Batch(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(cacheBucket))
			if bucket == nil {
				return fmt.Errorf("Cache bucket [%v] not found!", cacheBucket)
			}
			for _, anAuthor := range authors {
				marshalledAuthor, err := json.Marshal(anAuthor)
				if err != nil {
					return err
				}
				err = bucket.Put([]byte(anAuthor.UUID), marshalledAuthor)
				if err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			log.Errorf("ERROR storing to cache: %+v.", err)
		}
		wg.Done()
	}

	log.Info("Finished processing all authors.")
	if s.isInitialised() {
		s.setDataLoaded(true)
	}
}

func (s *authorServiceImpl) createCacheBucket() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		if tx.Bucket([]byte(cacheBucket)) != nil {
			log.Infof("Deleting bucket '%v'.", cacheBucket)
			if err := tx.DeleteBucket([]byte(cacheBucket)); err != nil {
				log.Warnf("Cache bucket [%v] could not be deleted.", cacheBucket)
			}
		}
		log.Infof("Creating bucket '%s'.", cacheBucket)
		_, err := tx.CreateBucket([]byte(cacheBucket))
		return err
	})
}

func (s *authorServiceImpl) getBerthaAuthors(berthaURL string) ([]berthaAuthor, error) {
	req, err := http.NewRequest("GET", berthaURL, nil)
	if err != nil {
		return []berthaAuthor{}, err
	}

	res, err := s.httpClient.Do(req)
	var bAuthors []berthaAuthor
	err = json.NewDecoder(res.Body).Decode(&bAuthors)
	return bAuthors, err
}

func (s *authorServiceImpl) loadCuratedAuthors(bAuthors []berthaAuthor) error {
	s.Lock()
	defer s.Unlock()
	err := s.db.Batch(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(cacheBucket))
		if bucket == nil {
			return fmt.Errorf("Cache bucket [%v] not found!", cacheBucket)
		}

		for _, b := range bAuthors {
			berthaUUID := uuid.NewMD5(uuid.UUID{}, []byte(b.TmeIdentifier)).String()
			cachedAuthor := bucket.Get([]byte(berthaUUID))
			var a author
			if cachedAuthor == nil {
				log.Warnf("Curated author %s [%s] was not found in cache.  Adding without V1 information.", b.Name, berthaUUID)
				a, _ = berthaToAuthor(b)
			} else {
				json.Unmarshal(cachedAuthor, &a)
				a, _ = addBerthaInformation(a, b)
			}

			newCachedVersion, _ := json.Marshal(a)
			bucket.Put([]byte(berthaUUID), newCachedVersion)
		}

		return nil
	})

	return err
}

func addBerthaInformation(a author, b berthaAuthor) (author, error) {
	berthaUUID := uuid.NewMD5(uuid.UUID{}, []byte(b.TmeIdentifier)).String()
	if berthaUUID != a.UUID {
		return a, errors.New("Bertha UUID doesn't match author UUID")
	}
	plainDescription, err := html2text.FromString(b.Biography)
	if err != nil {
		return a, err
	}
	a.Name = b.Name
	a.PrefLabel = b.Name
	a.EmailAddress = b.Email
	a.TwitterHandle = b.TwitterHandle
	a.FacebookProfile = b.FacebookProfile
	a.LinkedinProfile = b.LinkedinProfile
	a.Description = plainDescription
	a.DescriptionXML = b.Biography
	a.ImageURL = b.ImageURL

	return a, nil
}

func berthaToAuthor(a berthaAuthor) (author, error) {
	berthaUUID := uuid.NewMD5(uuid.UUID{}, []byte(a.TmeIdentifier)).String()
	plainDescription, err := html2text.FromString(a.Biography)

	if err != nil {
		return author{}, err
	}

	altIds := alternativeIdentifiers{
		UUIDs: []string{berthaUUID},
		TME:   []string{a.TmeIdentifier},
	}

	p := author{
		UUID:                   berthaUUID,
		Name:                   a.Name,
		PrefLabel:              a.Name,
		EmailAddress:           a.Email,
		TwitterHandle:          a.TwitterHandle,
		FacebookProfile:        a.FacebookProfile,
		LinkedinProfile:        a.LinkedinProfile,
		Description:            plainDescription,
		DescriptionXML:         a.Biography,
		ImageURL:               a.ImageURL,
		AlternativeIdentifiers: altIds,
	}

	return p, err
}
