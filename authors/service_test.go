package people

import (
	"bufio"
	"encoding/json"
	"github.com/Financial-Times/tme-reader/tmereader"
	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"
)

type testSuiteForPeople struct {
	name  string
	uuid  string
	found bool
	err   error
}

func TestInit(t *testing.T) {
	repo := blockingRepo{}
	repo.Add(1)
	tmpfile := getTempFile(t)
	defer os.Remove(tmpfile.Name())
	service := createTestPeopleService(&repo, tmpfile.Name())
	defer func() {
		repo.Done()
		service.Shutdown()
	}()
	assert.False(t, service.isDataLoaded())
	assert.True(t, service.isInitialised())
}

func TestGetPeople(t *testing.T) {
	tmpfile := getTempFile(t)
	defer os.Remove(tmpfile.Name())
	repo := dummyRepo{terms: []term{{CanonicalName: "Bob", RawID: "bob"}, {CanonicalName: "Fred", RawID: "fred"}}}
	service := createTestPeopleService(&repo, tmpfile.Name())
	defer service.Shutdown()
	waitTillInit(t, service)
	waitTillDataLoaded(t, service)
	pv, err := service.getPeople()

	var wg sync.WaitGroup
	var res []person
	wg.Add(1)
	go func(reader io.Reader, w *sync.WaitGroup) {
		var err error
		scan := bufio.NewScanner(reader)
		for scan.Scan() {
			var p person
			assert.NoError(t, err)
			err = json.Unmarshal(scan.Bytes(), &p)
			assert.NoError(t, err)
			res = append(res, p)
		}
		wg.Done()
	}(&pv, &wg)
	wg.Wait()

	assert.NoError(t, err)
	assert.Len(t, res, 2)
	assert.Equal(t, "28d66fcc-bb56-363d-80c1-f2d957ef58cf", res[0].UUID)
	assert.Equal(t, "be2e7e2b-0fa2-3969-a69b-74c46e754032", res[1].UUID)
}

func TestGetPeopleByUUID(t *testing.T) {
	tmpfile := getTempFile(t)
	defer os.Remove(tmpfile.Name())
	repo := dummyRepo{terms: []term{{CanonicalName: "Bob", RawID: "bob"}, {CanonicalName: "Fred", RawID: "fred"}}}
	service := createTestPeopleService(&repo, tmpfile.Name())
	defer service.Shutdown()
	waitTillInit(t, service)
	waitTillDataLoaded(t, service)
	pv, err := service.getPeopleUUIDs()

	var wg sync.WaitGroup
	var res []personUUID
	wg.Add(1)
	go func(reader io.Reader, w *sync.WaitGroup) {
		var err error
		scan := bufio.NewScanner(reader)
		for scan.Scan() {
			var p personUUID
			assert.NoError(t, err)
			err = json.Unmarshal(scan.Bytes(), &p)
			assert.NoError(t, err)
			res = append(res, p)
		}
		wg.Done()
	}(&pv, &wg)
	wg.Wait()

	assert.NoError(t, err)
	assert.Len(t, res, 2)
	assert.Equal(t, "28d66fcc-bb56-363d-80c1-f2d957ef58cf", res[0].UUID)
	assert.Equal(t, "be2e7e2b-0fa2-3969-a69b-74c46e754032", res[1].UUID)
}

func TestGetPeopleLink(t *testing.T) {
	tmpfile := getTempFile(t)
	defer os.Remove(tmpfile.Name())
	repo := dummyRepo{terms: []term{{CanonicalName: "Bob", RawID: "bob"}, {CanonicalName: "Fred", RawID: "fred"}}}
	service := createTestPeopleService(&repo, tmpfile.Name())
	defer service.Shutdown()
	waitTillInit(t, service)
	waitTillDataLoaded(t, service)
	pv, err := service.getPeopleLinks()

	var wg sync.WaitGroup
	var res []personLink
	wg.Add(1)
	go func(reader io.Reader, w *sync.WaitGroup) {
		var err error
		jsonBlob, err := ioutil.ReadAll(reader)
		assert.NoError(t, err)
		log.Infof("Got bytes: %v", string(jsonBlob[:]))
		err = json.Unmarshal(jsonBlob, &res)
		assert.NoError(t, err)
		wg.Done()
	}(&pv, &wg)
	wg.Wait()

	assert.NoError(t, err)
	assert.Len(t, res, 2)
	assert.Equal(t, "/base/url/28d66fcc-bb56-363d-80c1-f2d957ef58cf", res[0].APIURL)
	assert.Equal(t, "/base/url/be2e7e2b-0fa2-3969-a69b-74c46e754032", res[1].APIURL)
}

func TestGetCount(t *testing.T) {
	tmpfile := getTempFile(t)
	defer os.Remove(tmpfile.Name())
	repo := dummyRepo{terms: []term{{CanonicalName: "Bob", RawID: "bob"}, {CanonicalName: "Fred", RawID: "fred"}}}
	service := createTestPeopleService(&repo, tmpfile.Name())
	defer service.Shutdown()
	waitTillInit(t, service)
	waitTillDataLoaded(t, service)
	assertCount(t, service, 2)
}

func TestReload(t *testing.T) {
	tmpfile := getTempFile(t)
	defer os.Remove(tmpfile.Name())
	repo := dummyRepo{terms: []term{{CanonicalName: "Bob", RawID: "bob"}, {CanonicalName: "Fred", RawID: "fred"}}}
	service := createTestPeopleService(&repo, tmpfile.Name())
	defer service.Shutdown()
	waitTillInit(t, service)
	waitTillDataLoaded(t, service)
	assertCount(t, service, 2)
	repo.terms = append(repo.terms, term{CanonicalName: "Third", RawID: "third"})
	repo.count = 0
	assert.NoError(t, service.reloadDB())
	waitTillInit(t, service)
	waitTillDataLoaded(t, service)
	assertCount(t, service, 3)
}

func TestGetPersonByUUID(t *testing.T) {
	tmpfile := getTempFile(t)
	defer os.Remove(tmpfile.Name())
	repo := dummyRepo{terms: []term{{CanonicalName: "Bob", RawID: "bob"}, {CanonicalName: "Fred", RawID: "fred"}}}
	service := createTestPeopleService(&repo, tmpfile.Name())
	defer service.Shutdown()
	waitTillInit(t, service)
	waitTillDataLoaded(t, service)

	tests := []testSuiteForPeople{
		{"Success", "28d66fcc-bb56-363d-80c1-f2d957ef58cf", true, nil},
		{"Success", "xxxxxxxx-bb56-363d-80c1-f2d957ef58cf", false, nil}}
	for _, test := range tests {
		person, found, err := service.getPersonByUUID(test.uuid)
		if test.err != nil {
			assert.Equal(t, test.err, err)
		} else if test.found {
			assert.True(t, found)
			assert.NotNil(t, person)
		} else {
			assert.False(t, found)
		}
	}
}

func TestFailingOpeningDB(t *testing.T) {
	dir, err := ioutil.TempDir("", "service_test")
	assert.NoError(t, err)
	service := createTestPeopleService(&dummyRepo{}, dir)
	defer service.Shutdown()
	for i := 1; i <= 1000; i++ {
		if !service.isInitialised() {
			log.Info("isInitialised was false")
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	assert.False(t, service.isInitialised(), "isInitialised should be false")
}

func assertCount(t *testing.T, s PeopleService, expected int) {
	count, err := s.getCount()
	assert.NoError(t, err)
	assert.Equal(t, expected, count)
}

func createTestPeopleService(repo tmereader.Repository, cacheFileName string) PeopleService {
	return NewPeopleService(repo, "/base/url", "taxonomy_string", 1, cacheFileName)
}

func getTempFile(t *testing.T) *os.File {
	tmpfile, err := ioutil.TempFile("", "example")
	assert.NoError(t, err)
	assert.NoError(t, tmpfile.Close())
	log.Debug("File:%s", tmpfile.Name())
	return tmpfile
}

func waitTillInit(t *testing.T, s PeopleService) {
	for i := 1; i <= 1000; i++ {
		if s.isInitialised() {
			log.Info("isInitialised was true")
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	assert.True(t, s.isInitialised())
}

func waitTillDataLoaded(t *testing.T, s PeopleService) {
	for i := 1; i <= 1000; i++ {
		if s.isDataLoaded() {
			log.Info("isDataLoaded was true")
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	assert.True(t, s.isDataLoaded())
}

type dummyRepo struct {
	sync.Mutex
	terms []term
	err   error
	count int
}

func (d *dummyRepo) GetTmeTermsFromIndex(startRecord int) ([]interface{}, error) {
	defer func() {
		d.count++
	}()
	if len(d.terms) == d.count {
		return nil, d.err
	}
	return []interface{}{d.terms[d.count]}, d.err
}

// Never used
func (d *dummyRepo) GetTmeTermById(uuid string) (interface{}, error) {
	return nil, nil
}

type blockingRepo struct {
	sync.WaitGroup
	err  error
	done bool
}

func (d *blockingRepo) GetTmeTermsFromIndex(startRecord int) ([]interface{}, error) {
	d.Wait()
	if d.done {
		return nil, d.err
	}
	d.done = true
	return []interface{}{term{CanonicalName: "Bob", RawID: "bob"}}, d.err
}

// Never used
func (d *blockingRepo) GetTmeTermById(uuid string) (interface{}, error) {
	return nil, nil
}
