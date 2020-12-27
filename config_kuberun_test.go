package kubernetes_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/containerssh/kubernetes"
)

// TestUnmarshalYAML03 tests the ContainerSSH 0.3 compatibility. It checks if a config fragment from 0.3 can still be
// unmarshalled.
func TestUnmarshalYAML03(t *testing.T) {
	testFile, err := os.Open("testdata/config-0.3.yaml")
	assert.NoError(t, err)
	unmarshaller := yaml.NewDecoder(testFile)
	unmarshaller.KnownFields(true)
	//goland:noinspection GoDeprecation
	config := kubernetes.KubeRunConfig{}
	assert.NoError(t, unmarshaller.Decode(&config))
}
