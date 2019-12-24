package proc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSetDefault(t *testing.T) {
	p := Processor{
		Conf: &Conf{},
	}

	assert.Equal(t, p.Conf.System.Concurrent, 0)
	assert.Equal(t, p.Conf.System.MaxItems, 0)
	assert.Equal(t, p.Conf.System.MaxTotal, 0)
	assert.Equal(t, p.Conf.System.MaxKeepInDB, 0)
	assert.Equal(t, p.Conf.System.UpdateInterval, time.Duration(0))

	p.setDefaults()

	assert.Equal(t, p.Conf.System.Concurrent, 8)
	assert.Equal(t, p.Conf.System.MaxItems, 5)
	assert.Equal(t, p.Conf.System.MaxTotal, 100)
	assert.Equal(t, p.Conf.System.MaxKeepInDB, 5000)
	assert.Equal(t, p.Conf.System.UpdateInterval, time.Minute*5)
}
