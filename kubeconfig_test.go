package kubernetes

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os/user"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func setConfigFromKubeConfig(config *Config) (err error) {
	usr, err := user.Current()
	if err != nil {
		return err
	}
	kubectlConfig, err := readKubeConfig(filepath.Join(usr.HomeDir, ".kube", "config"))
	if err != nil {
		return fmt.Errorf("failed to read kubeconfig (%w)", err)
	}
	context := extractKubeConfigContext(kubectlConfig, kubectlConfig.CurrentContext)
	if context == nil {
		return fmt.Errorf("failed to find current context in kubeConfig")
	}

	kubeConfigUser := extractKubeConfigUser(kubectlConfig, context.Context.User)
	if kubeConfigUser == nil {
		return fmt.Errorf("failed to find user in kubeConfig")
	}

	kubeConfigCluster := extractKubeConfigCluster(kubectlConfig, context.Context.Cluster)
	if kubeConfigCluster == nil {
		return fmt.Errorf("failed to find cluster in kubeConfig")
	}

	config.Connection.Host = strings.Replace(
		kubeConfigCluster.Cluster.Server,
		"https://",
		"",
		1,
	)
	if err = configureCertificates(kubeConfigCluster, kubeConfigUser, config); err != nil {
		return err
	}

	return nil
}

func extractKubeConfigContext(kubectlConfig kubeConfig, currentContext string) *kubeConfigContext {
	var kubeContext *kubeConfigContext
	for _, ctx := range kubectlConfig.Contexts {
		if ctx.Name == currentContext {
			currentKubeContext := ctx
			kubeContext = &currentKubeContext
			break
		}
	}
	return kubeContext
}

func configureCertificates(
	kubeConfigCluster *kubeConfigCluster,
	kubeConfigUser *kubeConfigUser,
	config *Config,
) error {
	decodedCa, err := base64.StdEncoding.DecodeString(
		kubeConfigCluster.Cluster.CertificateAuthorityData,
	)
	if err != nil {
		return err
	}
	config.Connection.CAData = string(decodedCa)

	if kubeConfigUser.User.ClientKeyData != "" {
		decodedKey, err := base64.StdEncoding.DecodeString(
			kubeConfigUser.User.ClientKeyData,
		)
		if err != nil {
			return err
		}
		config.Connection.KeyData = string(decodedKey)
	}

	if kubeConfigUser.User.ClientCertificateData != "" {
		decodedCert, err := base64.StdEncoding.DecodeString(
			kubeConfigUser.User.ClientCertificateData,
		)
		if err != nil {
			return err
		}
		config.Connection.CertData = string(decodedCert)
	}

	config.Connection.BearerToken = kubeConfigUser.User.Token
	return nil
}

func extractKubeConfigCluster(kubectlConfig kubeConfig, clusterName string) *kubeConfigCluster {
	var kubeConfigCluster *kubeConfigCluster
	for _, c := range kubectlConfig.Clusters {
		if c.Name == clusterName {
			currentKubeConfigCluster := c
			kubeConfigCluster = &currentKubeConfigCluster
			break
		}
	}
	return kubeConfigCluster
}

func extractKubeConfigUser(kubectlConfig kubeConfig, userName string) *kubeConfigUser {
	var kubeConfigUser *kubeConfigUser
	for _, u := range kubectlConfig.Users {
		if u.Name == userName {
			currentConfigUser := u
			kubeConfigUser = &currentConfigUser
			break
		}
	}
	return kubeConfigUser
}

type kubeConfig struct {
	ApiVersion     string              `yaml:"apiVersion" default:"v1"`
	Clusters       []kubeConfigCluster `yaml:"clusters"`
	Contexts       []kubeConfigContext `yaml:"contexts"`
	CurrentContext string              `yaml:"current-context"`
	Kind           string              `yaml:"kind" default:"Config"`
	Preferences    map[string]string   `yaml:"preferences"`
	Users          []kubeConfigUser    `yaml:"users"`
}

type kubeConfigCluster struct {
	Name    string `yaml:"name"`
	Cluster struct {
		CertificateAuthorityData string `yaml:"certificate-authority-data"`
		Server                   string `yaml:"server"`
	} `yaml:"cluster"`
}

type kubeConfigContext struct {
	Name    string `yaml:"name"`
	Context struct {
		Cluster string `yaml:"cluster"`
		User    string `yaml:"user"`
	} `yaml:"context"`
}

type kubeConfigUser struct {
	Name string `yaml:"name"`
	User struct {
		ClientCertificateData string `yaml:"client-certificate-data"`
		ClientKeyData         string `yaml:"client-key-data"`
		Token                 string `yaml:"token"`
	} `yaml:"user"`
}

func readKubeConfig(file string) (config kubeConfig, err error) {
	yamlFile, err := ioutil.ReadFile(file)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}
