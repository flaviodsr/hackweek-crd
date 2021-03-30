package kfserving

import (
	"context"

	kfservingv1 "github.com/kubeflow/kfserving/pkg/apis/serving/v1beta1"
	"github.com/pkg/errors"
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

var log = logf.Log.WithName("KFServingReconciler")

type KfservingReconciler struct {
	client  client.Client
	scheme  *runtime.Scheme
	Service *kfservingv1.InferenceService
}

func NewKfservingReconciler(client client.Client,
	scheme *runtime.Scheme,
	componentMeta metav1.ObjectMeta,
	isvcSpec *kfservingv1.InferenceServiceSpec) *KfservingReconciler {
	return &KfservingReconciler{
		client:  client,
		scheme:  scheme,
		Service: createKfservingService(componentMeta, isvcSpec),
	}
}

func createKfservingService(componentMeta metav1.ObjectMeta, isvcSpec *kfservingv1.InferenceServiceSpec) *kfservingv1.InferenceService {
	service := &kfservingv1.InferenceService{
		ObjectMeta: metav1.ObjectMeta{
			Name:        componentMeta.Name,
			Namespace:   componentMeta.Namespace,
			Labels:      componentMeta.Labels,
			Annotations: componentMeta.Annotations,
		},
		Spec: *isvcSpec,
	}
	return service
}

func (r *KfservingReconciler) Reconcile() (*kfservingv1.InferenceServiceStatus, error) {
	// Create service if does not exist
	desired := r.Service
	existing := &kfservingv1.InferenceService{}

	err := r.client.Get(context.TODO(), types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
	if err != nil {
		if apierr.IsNotFound(err) {
			log.Info("Creating knative service", "namespace", desired.Namespace, "name", desired.Name)
			return &desired.Status, r.client.Create(context.TODO(), desired)
		}
		return nil, err
	}
	// Return if no differences to reconcile.
	if semanticEquals(desired, existing) {
		return &existing.Status, nil
	}

	// Reconcile differences and update
	diff, err := kmp.SafeDiff(desired.Spec.Predictor, existing.Spec.Predictor)
	if err != nil {
		return &existing.Status, errors.Wrapf(err, "failed to diff knative service configuration spec")
	}
	log.Info("kfserving inference service configuration diff (-desired, +observed):", "diff", diff)
	existing.Spec.Predictor = desired.Spec.Predictor
	existing.ObjectMeta.Labels = desired.ObjectMeta.Labels
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		log.Info("Updating kfserving service", "namespace", desired.Namespace, "name", desired.Name)
		return r.client.Update(context.TODO(), existing)
	})
	if err != nil {
		return &existing.Status, errors.Wrapf(err, "fails to update knative service")
	}
	return &existing.Status, nil

}

func semanticEquals(desiredService, service *kfservingv1.InferenceService) bool {
	return equality.Semantic.DeepEqual(desiredService.Spec.Predictor, service.Spec.Predictor) &&
		equality.Semantic.DeepEqual(desiredService.ObjectMeta.Labels, service.ObjectMeta.Labels)
}
