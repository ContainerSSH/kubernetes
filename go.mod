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
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a // indirect
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c // indirect
	golang.org/x/term v0.0.0-20210503060354-a79de5458b56 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	k8s.io/api v0.22.3
	k8s.io/apimachinery v0.22.3
	k8s.io/client-go v0.22.3
	sigs.k8s.io/yaml v1.2.0
)

replace github.com/containerssh/log v1.1.3 => ../log
