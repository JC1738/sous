package sous

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuildResultString(t *testing.T) {
	assert := assert.New(t)

	br := &BuildResult{
		Elapsed:     time.Second * 5,
		Advisories:  Advisories{EphemeralTag},
		VersionName: "something-something-2.3.4",
	}

	str := fmt.Sprintln(br)
	assert.Regexp(`ephemeral`, str)
	assert.Regexp(`2.3.4`, str)
	assert.Regexp(`5s`, str)
}
