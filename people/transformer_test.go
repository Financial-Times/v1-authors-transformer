package people

import (
	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
	"github.com/pborman/uuid"
	"fmt"
)

const (
	taxonomyName = "taxonomy_name"
)

func TestTransformPerson(t *testing.T) {
	testTerm := term{
		CanonicalName: "Bob",
		RawID:         "bob",
		Aliases: aliases{
			Alias: []alias{
				{Name: "B"},
				{Name: "b"},
			}},
	}
	tfp := transformPerson(testTerm, taxonomyName)
	log.Infof("got person %v", tfp)
	assert.NotNil(t, tfp)
	assert.Len(t, tfp.Aliases, 2)
	assert.Equal(t, "B", tfp.Aliases[0])
	assert.Equal(t, "b", tfp.Aliases[1])
	assert.Equal(t, "0e86d39b-8320-3a98-a87a-ff35d2cb04b9", tfp.UUID)
	assert.Equal(t, "Bob", tfp.PrefLabel)
}

func TestTmeId(t *testing.T) {
	fmt.Printf("JULIA \n")
	tmeIdentifier := "TnN0ZWluX1BOX1BvbGl0aWNpYW5fNDA1-UE4=";
	personUUID := uuid.NewMD5(uuid.UUID{}, []byte(tmeIdentifier)).String()
	fmt.Printf(">>>>>> %s \n", personUUID)

	testTerm := term{
		CanonicalName: "Bob",
		RawID:         "bob",
		Aliases: aliases{
			Alias: []alias{
				{Name: "B"},
				{Name: "b"},
			}},
	}
	tfp := transformPerson(testTerm, taxonomyName)
	log.Infof("got person %v", tfp)
	assert.NotNil(t, tfp)
	assert.Len(t, tfp.Aliases, 2)
	assert.Equal(t, "B", tfp.Aliases[0])
	assert.Equal(t, "b", tfp.Aliases[1])
	assert.Equal(t, "0e86d39b-8320-3a98-a87a-ff35d2cb04b9", tfp.UUID)
	assert.Equal(t, "Bob", tfp.PrefLabel)
}