package snowflake_test

import (
	"errors"
	"os"
	"testing"

	"github.com/ttlj/snowflake"
)

func TestPodNameWorkerID(t *testing.T) {
	var parameters = map[string]struct {
		result uint16
		err    error
	}{
		"pod-0":                   {0, nil},
		"name-name-1":             {1, nil},
		"name-5-name-2":           {2, nil},
		"name-5-name-99999":       {0, errors.New("error")},
		"ingress-549d59d75-2wngg": {0, errors.New("error")},
	}

	for input, test := range parameters {
		os.Setenv("MY_POD_NAME", input)
		actual, err := snowflake.K8sPodID()
		if actual != test.result && err == nil {
			t.Errorf("Input: %s; expected: %d: , actual: %d",
				input, test.result, actual)
		}
		if err == nil && (test.err != nil) {
			t.Errorf("Expected error, got nil")
		}
	}
}

func TestPodEnvVarIPWorkerID(t *testing.T) {
	var parameters = map[string]struct {
		result uint16
		err    error
	}{
		"1.2.3.4":     {772, nil},
		"234.45.56.7": {14343, nil},
		"1.2.3":       {0, errors.New("error")},
		"1.2.3.4.5":   {0, errors.New("error")},
		"a.b.c.d":     {0, errors.New("error")},
	}

	for input, test := range parameters {
		os.Setenv("MY_HOST_IP", input)
		actual, err := snowflake.EnvVarIPWorkerID()
		if actual != test.result && err == nil {
			t.Errorf("Input: %s; expected: %d: , actual: %d",
				input, test.result, actual)
		}
		if err == nil && (test.err != nil) {
			t.Errorf("Expected error, got nil")
		}
	}
}
