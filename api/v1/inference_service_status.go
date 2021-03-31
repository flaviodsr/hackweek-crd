/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	kfservingv1 "github.com/kubeflow/kfserving/pkg/apis/serving/v1beta1"
	seldonv1 "github.com/seldonio/seldon-core/operator/apis/machinelearning.seldon.io/v1"
	"knative.dev/pkg/apis"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// InferenceServiceStatus defines the observed state of InferenceService
type InferenceServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Status StatusState `json:"state,omitempty" protobuf:"string,1,opt,name=state"`

	// URL holds the url that will distribute traffic over the provided traffic targets.
	// It generally has the form http[s]://{route-name}.{route-namespace}.{cluster-level-suffix}
	// +optional
	URL *apis.URL `json:"url,omitempty"`
}

type StatusState string

// CRD Status values
const (
	StatusStateAvailable StatusState = "Available"
	StatusStateCreating  StatusState = "Creating"
	StatusStateFailed    StatusState = "Failed"
)

func (ss *InferenceServiceStatus) PropagateStatusFromKfserving(serviceStatus *kfservingv1.InferenceServiceStatus) {
	// propagate overall service condition
	if len(serviceStatus.Status.Conditions) <= 0 {
		ss.Status = StatusStateCreating
	} else {
		status := serviceStatus.Status.Conditions[0].Status
		switch status {
		case "True":
			if serviceStatus.Address != nil {
				ss.Status = StatusStateAvailable
				ss.URL = serviceStatus.URL
			} else {
				ss.Status = StatusStateCreating
			}
		case "False":
			ss.Status = StatusStateFailed
		default:
			ss.Status = StatusStateCreating
		}
	}
}

func (ss *InferenceServiceStatus) PropagateStatusFromSeldon(serviceStatus *seldonv1.SeldonDeploymentStatus) {
	switch serviceStatus.State {
	case seldonv1.StatusStateAvailable:
		if serviceStatus.Address != nil {
			url, _ := apis.ParseURL(serviceStatus.Address.URL)
			ss.URL = url
		}
		ss.Status = StatusStateAvailable
	case seldonv1.StatusStateFailed:
		ss.Status = StatusStateFailed
	default:
		ss.Status = StatusStateCreating
	}
}
