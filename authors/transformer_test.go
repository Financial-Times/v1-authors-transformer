package authors

import (
	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
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
