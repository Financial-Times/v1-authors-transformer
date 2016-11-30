package people

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Financial-Times/tme-reader/tmereader"
	log "github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"io"
	"sync"
	"time"
)

const (
	cacheBucket = "person"
	//uppAuthority = "http://api.ft.com/system/FT-UPP"
	//tmeAuthority = "http://api.ft.com/system/FT-TME"
)

type PeopleService interface {
	getPeople() (io.PipeReader, error)
	getPeopleLinks() (io.PipeReader, error)
	getPeopleUUIDs() (io.PipeReader, error)
	getPersonByUUID(uuid string) (person, bool, error)
	getCount() (int, error)
	isInitialised() bool
	isDataLoaded() bool
	reloadDB() error
	Shutdown() error
}

type peopleServiceImpl struct {
	sync.RWMutex
	repository    tmereader.Repository
	baseURL       string
	taxonomyName  string
	maxTmeRecords int
	initialised   bool
	dataLoaded    bool
	cacheFileName string
	db            *bolt.DB
}

func NewPeopleService(repo tmereader.Repository, baseURL string, taxonomyName string, maxTmeRecords int, cacheFileName string) PeopleService {
	s := &peopleServiceImpl{repository: repo, baseURL: baseURL, taxonomyName: taxonomyName, maxTmeRecords: maxTmeRecords, initialised: true, cacheFileName: cacheFileName}
	go func(service *peopleServiceImpl) {
		err := service.loadDB()
		if err != nil {
			log.Errorf("Error while creating PeopleService: [%v]", err.Error())
		}
	}(s)
	return s
}

func (s *peopleServiceImpl) isInitialised() bool {
	s.RLock()
	defer s.RUnlock()
	return s.initialised
}

func (s *peopleServiceImpl) setInitialised(val bool) {
	s.Lock()
	s.initialised = val
	s.Unlock()
}

func (s *peopleServiceImpl) isDataLoaded() bool {
	s.RLock()
	defer s.RUnlock()
	return s.dataLoaded
}

func (s *peopleServiceImpl) setDataLoaded(val bool) {
	s.Lock()
	s.dataLoaded = val
	s.Unlock()
}

func (s *peopleServiceImpl) Shutdown() error {
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

func (s *peopleServiceImpl) getCount() (int, error) {
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

func (s *peopleServiceImpl) getPeople() (io.PipeReader, error) {
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

func (s *peopleServiceImpl) getPeopleUUIDs() (io.PipeReader, error) {
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
				pl := personUUID{UUID: string(k[:])}
				if err := encoder.Encode(pl); err != nil {
					return err
				}
			}
			return nil
		})
	}()
	return *pv, nil
}

func (s *peopleServiceImpl) getPeopleLinks() (io.PipeReader, error) {
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
				pl := personLink{APIURL: s.baseURL + "/" + string(k[:])}
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

func (s *peopleServiceImpl) getPersonByUUID(uuid string) (person, bool, error) {
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
		return person{}, false, err
	}
	if len(cachedValue) == 0 {
		log.Infof("INFO No cached value for [%v].", uuid)
		return person{}, false, nil
	}

	var cachedPerson person
	if err := json.Unmarshal(cachedValue, &cachedPerson); err != nil {
		log.Errorf("ERROR unmarshalling cached value for [%v]: %v.", uuid, err.Error())
		return person{}, true, err
	}
	return cachedPerson, true, nil
}

func (s *peopleServiceImpl) openDB() error {
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

func (s *peopleServiceImpl) reloadDB() error {
	s.setDataLoaded(false)
	return s.loadDB()
}

func (s *peopleServiceImpl) loadDB() error {
	var wg sync.WaitGroup
	log.Info("Loading DB...")
	c := make(chan []person)
	go s.processPeople(c, &wg)
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
			log.Info("Finished fetching people from TME. Waiting subroutines to terminate.")
			break
		}

		wg.Add(1)
		s.processTerms(terms, c)
		responseCount += s.maxTmeRecords
	}
	return nil
}

func (s *peopleServiceImpl) processTerms(terms []interface{}, c chan<- []person) {
	log.Info("Processing terms...")
	var cacheToBeWritten []person
	for _, iTerm := range terms {
		t := iTerm.(term)
		cacheToBeWritten = append(cacheToBeWritten, transformPerson(t, s.taxonomyName))
	}
	c <- cacheToBeWritten
}

func (s *peopleServiceImpl) processPeople(c <-chan []person, wg *sync.WaitGroup) {
	for people := range c {
		log.Infof("Processing batch of %v people.", len(people))
		if err := s.db.Batch(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(cacheBucket))
			if bucket == nil {
				return fmt.Errorf("Cache bucket [%v] not found!", cacheBucket)
			}
			for _, anPerson := range people {
				marshalledPerson, err := json.Marshal(anPerson)
				if err != nil {
					return err
				}
				err = bucket.Put([]byte(anPerson.UUID), marshalledPerson)
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

	log.Info("Finished processing all people.")
	if s.isInitialised() {
		s.setDataLoaded(true)
	}
}

func (s *peopleServiceImpl) createCacheBucket() error {
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
