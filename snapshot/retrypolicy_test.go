package snapshot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_parseRetryPolicyBackoff(t *testing.T) {
	for _, testcase := range []struct {
		Input      string
		ExpectBase time.Duration
		ExpectMax  time.Duration
		ExpectErr  bool
	}{
		{
			Input: "",
		},
		{
			Input:      "10m",
			ExpectBase: 10 * time.Minute,
		},
		{
			Input:      "5s,10m",
			ExpectBase: 5 * time.Second,
			ExpectMax:  10 * time.Minute,
		},
		{
			Input:     "invalid",
			ExpectErr: true,
		},
		{
			Input:     "5s,10m,20m",
			ExpectErr: true,
		},
	} {
		t.Run(testcase.Input, func(t *testing.T) {
			out, err := parseRetryPolicyBackoff(testcase.Input)
			if testcase.ExpectErr { //nolint:gocritic
				assert.Error(t, err)
			} else if testcase.ExpectBase == 0 && testcase.ExpectMax == 0 {
				assert.Nil(t, out)
			} else {
				assert.Equal(t, testcase.ExpectBase, out.BaseInterval.AsDuration())
				assert.Equal(t, testcase.ExpectMax, out.MaxInterval.AsDuration())
			}
		})
	}
}
