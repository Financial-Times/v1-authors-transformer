package authors

import (
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	taxonomyName = "taxonomy_name"
)

func TestTransformAuthor(t *testing.T) {
	testTerm := term{
		CanonicalName: "Bob",
		RawID:         "bob",
		Aliases: aliases{
			Alias: []alias{
				{Name: "B"},
				{Name: "b"},
			}},
	}
	tfp := transformAuthor(testTerm, taxonomyName)
	log.Infof("got author %v", tfp)
	assert.NotNil(t, tfp)
	assert.EqualValues(t, []string{"B", "b", "Bob"}, tfp.Aliases)
	assert.Equal(t, "0e86d39b-8320-3a98-a87a-ff35d2cb04b9", tfp.UUID)
	assert.Equal(t, "Bob", tfp.PrefLabel)
}
