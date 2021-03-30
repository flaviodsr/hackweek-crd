module fuseml.suse

go 1.13

require (
	github.com/aws/aws-sdk-go v1.34.18 // indirect
	github.com/go-logr/logr v0.3.0
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/kubeflow/kfserving v0.5.1
	github.com/onsi/ginkgo v1.15.0
	github.com/onsi/gomega v1.10.4
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.6.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v12.0.0+incompatible
	knative.dev/pkg v0.0.0-20200922164940-4bf40ad82aab
	knative.dev/serving v0.18.0
	sigs.k8s.io/controller-runtime v0.7.0
)

replace k8s.io/client-go => k8s.io/client-go v0.19.2
