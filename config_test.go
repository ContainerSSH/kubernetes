package kubernetes_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/containerssh/kubernetes"
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
