module github.com/containerssh/kubernetes/v2

go 1.16

require (
	github.com/containerssh/geoip v1.0.0
	github.com/containerssh/http v1.1.0 // indirect
	github.com/containerssh/log v1.1.6
	github.com/containerssh/metrics v1.0.0
	github.com/containerssh/sshserver v1.0.0
	github.com/containerssh/structutils v1.0.0
	github.com/containerssh/unixutils v1.0.0
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/go-openapi/spec v0.19.3 // indirect
	github.com/google/go-cmp v0.5.5 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/stretchr/testify v1.7.0
	google.golang.org/appengine v1.6.7 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	k8s.io/api v0.23.0
	k8s.io/apimachinery v0.23.0
	k8s.io/client-go v0.23.0
	sigs.k8s.io/yaml v1.2.0
)

replace github.com/containerssh/log v1.1.3 => ../log
