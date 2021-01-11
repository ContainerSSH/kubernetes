module github.com/containerssh/kubernetes

go 1.14

require (
	github.com/containerssh/geoip v0.9.3
	github.com/containerssh/log v0.9.8
	github.com/containerssh/metrics v0.9.5
	github.com/containerssh/service v0.9.0
	github.com/containerssh/sshserver v0.9.14
	github.com/containerssh/structutils v0.9.0
	github.com/containerssh/unixutils v0.9.0
	github.com/creasty/defaults v1.5.1
	github.com/stretchr/testify v1.6.1
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
	k8s.io/api v0.20.1
	k8s.io/apimachinery v0.20.1
	k8s.io/client-go v0.20.1
)

replace github.com/containerssh/sshserver v0.9.14 => ../sshserver
