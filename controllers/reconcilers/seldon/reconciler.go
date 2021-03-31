package seldon

import (
	"context"

	"github.com/pkg/errors"
	seldonv1 "github.com/seldonio/seldon-core/operator/apis/machinelearning.seldon.io/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"knative.dev/pkg/kmp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("SeldonReconciler")

type SeldonReconciler struct {
	client  client.Client
	scheme  *runtime.Scheme
	Service *seldonv1.SeldonDeployment
}

func NewSeldonReconciler(client client.Client,
	scheme *runtime.Scheme,
	componentMeta metav1.ObjectMeta,
	sDeploymentSpec *seldonv1.SeldonDeploymentSpec) *SeldonReconciler {
	return &SeldonReconciler{
		client:  client,
		scheme:  scheme,
		Service: createSeldonService(componentMeta, sDeploymentSpec),
	}
}

func createSeldonService(componentMeta metav1.ObjectMeta, sDeploymentSpec *seldonv1.SeldonDeploymentSpec) *seldonv1.SeldonDeployment {
	service := &seldonv1.SeldonDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        componentMeta.Name,
			Namespace:   componentMeta.Namespace,
			Labels:      componentMeta.Labels,
			Annotations: componentMeta.Annotations,
		},
		Spec: *sDeploymentSpec,
	}
	return service
}

func (r *SeldonReconciler) Reconcile() (*seldonv1.SeldonDeploymentStatus, error) {
	// Create service if does not exist
	desired := r.Service
	existing := &seldonv1.SeldonDeployment{}

	err := r.client.Get(context.TODO(), types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
	if err != nil {
		if apierr.IsNotFound(err) {
			log.Info("Creating seldon deployment", "namespace", desired.Namespace, "name", desired.Name)
			return &desired.Status, r.client.Create(context.TODO(), desired)
		}
		return nil, err
	}
	// Return if no differences to reconcile.
	if semanticEquals(desired, existing) {
		return &existing.Status, nil
	}

	// Reconcile differences and update
	diff, err := kmp.SafeDiff(desired.Spec.Predictors, existing.Spec.Predictors)
	if err != nil {
		return &existing.Status, errors.Wrapf(err, "failed to diff sledon deplyoment configuration spec")
	}
	log.Info("seldon deployment configuration diff (-desired, +observed):", "diff", diff)
	existing.Spec.Predictors = desired.Spec.Predictors
	existing.ObjectMeta.Labels = desired.ObjectMeta.Labels
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		log.Info("Updating seldon deployment", "namespace", desired.Namespace, "name", desired.Name)
		return r.client.Update(context.TODO(), existing)
	})
	if err != nil {
		return &existing.Status, errors.Wrapf(err, "fails to update seldon deployment")
	}
	return &existing.Status, nil

}

func semanticEquals(desiredService, service *seldonv1.SeldonDeployment) bool {
	return equality.Semantic.DeepEqual(desiredService.Spec.Predictors, service.Spec.Predictors) &&
		equality.Semantic.DeepEqual(desiredService.ObjectMeta.Labels, service.ObjectMeta.Labels)
}
