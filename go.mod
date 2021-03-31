module fuseml.suse

go 1.13

require (
	github.com/go-logr/logr v0.3.0
	github.com/go-logr/zapr v0.2.0 // indirect
	github.com/googleapis/gnostic v0.5.1 // indirect
	github.com/kubeflow/kfserving v0.5.1
	github.com/onsi/ginkgo v1.15.0
	github.com/onsi/gomega v1.10.4
	github.com/pkg/errors v0.9.1
	github.com/seldonio/seldon-core/operator v0.0.0-20210329163018-5939e9bdbcf4
	k8s.io/api v0.19.2
	k8s.io/apiextensions-apiserver v0.19.2 // indirect
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/utils v0.0.0-20200912215256-4140de9c8800 // indirect
	knative.dev/pkg v0.0.0-20200922164940-4bf40ad82aab
	knative.dev/serving v0.18.0
	sigs.k8s.io/controller-runtime v0.7.0
)

replace (
	k8s.io/client-go => k8s.io/client-go v0.19.2
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.6.5
)
