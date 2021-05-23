package kubernetes_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/containerssh/structutils"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/containerssh/kubernetes/v2"
)

func TestLoadIssue209(t *testing.T) {
	testFile, err := os.Open("testdata/issue-209.yaml")
	assert.NoError(t, err)
	config := kubernetes.Config{}
	unmarshaller := yaml.NewDecoder(testFile)
	unmarshaller.KnownFields(true)
	assert.NoError(t, unmarshaller.Decode(&config))

	assert.Equal(t, "/home/ubuntu", config.Pod.Spec.Volumes[0].HostPath.Path)
}

func TestLoadSave(t *testing.T) {
	oldConfig := &kubernetes.Config{}
	newConfig := &kubernetes.Config{}

	structutils.Defaults(oldConfig)

	data := &bytes.Buffer{}
	encoder := yaml.NewEncoder(data)
	if err := encoder.Encode(oldConfig); err != nil {
		t.Fatal(err)
	}
	decoder := yaml.NewDecoder(data)
	if err := decoder.Decode(newConfig); err != nil {
		t.Fatal(err)
	}

	diff := cmp.Diff(
		oldConfig,
		newConfig,
		cmp.AllowUnexported(kubernetes.PodConfig{}),
		cmp.AllowUnexported(kubernetes.ConnectionConfig{}),
		cmpopts.EquateEmpty(),
	)
	if diff != "" {
		t.Fatal(fmt.Errorf("restored configuration is different from the saved config: %v", diff))
	}
}