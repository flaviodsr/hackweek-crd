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

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kfservingv1 "github.com/kubeflow/kfserving/pkg/apis/serving/v1beta1"
	kfservingv1const "github.com/kubeflow/kfserving/pkg/constants"
	seldonv1 "github.com/seldonio/seldon-core/operator/apis/machinelearning.seldon.io/v1"
	seldonv1const "github.com/seldonio/seldon-core/operator/constants"

	servingv1 "fuseml.suse/api/v1"
	"fuseml.suse/controllers/reconcilers/kfserving"
	"fuseml.suse/controllers/reconcilers/seldon"
	"fuseml.suse/controllers/utils"
)

// InferenceServiceReconciler reconciles a InferenceService object
type InferenceServiceReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=serving.fuseml.suse,resources=inferenceservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=serving.fuseml.suse,resources=inferenceservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=serving.kubeflow.org,resources=inferenceservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=serving.kubeflow.org,resources=inferenceservices/status,verbs=get
// +kubebuilder:rbac:groups=machinelearning.seldon.io,resources=seldondeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=machinelearning.seldon.io,resources=seldondeployments/status,verbs=get

func (r *InferenceServiceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("inferenceservice", req.NamespacedName)

	// Fetch the InferenceService instance
	infSvc := &servingv1.InferenceService{}
	if err := r.Get(ctx, req.NamespacedName, infSvc); err != nil {
		if apierr.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// name of our custom finalizer
	finalizerName := "fuseml.inferenceservice.finalizers"

	// examine DeletionTimestamp to determine if object is under deletion
	if infSvc.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !utils.ContainsString(infSvc.ObjectMeta.Finalizers, finalizerName) {
			infSvc.ObjectMeta.Finalizers = append(infSvc.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(context.Background(), infSvc); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if utils.ContainsString(infSvc.ObjectMeta.Finalizers, finalizerName) {
			// remove our finalizer from the list and update it.
			infSvc.ObjectMeta.Finalizers = utils.RemoveString(infSvc.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(context.Background(), infSvc); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}
	log.Info("Reconciling inference service", "apiVersion", infSvc.APIVersion, "isvc", infSvc.Name)

	objectMeta := metav1.ObjectMeta{
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
		Name:        infSvc.Name,
		Namespace:   infSvc.Namespace,
	}
	for k, v := range infSvc.ObjectMeta.Annotations {
		objectMeta.Annotations[k] = v
	}
	for k, v := range infSvc.ObjectMeta.Labels {
		objectMeta.Labels[k] = v
	}

	if infSvc.Spec.Backend == "kfserving" {
		timeoutSeconds := int64(60)
		defaultProtocol := kfservingv1const.ProtocolV2
		runtimeVersion := "0.2.1"
		spec := kfservingv1.InferenceServiceSpec{
			Predictor: kfservingv1.PredictorSpec{
				ComponentExtensionSpec: kfservingv1.ComponentExtensionSpec{
					TimeoutSeconds: &timeoutSeconds,
				},
				PodSpec: kfservingv1.PodSpec{
					ServiceAccountName: infSvc.Spec.ServiceAccountName,
				},
				SKLearn: &kfservingv1.SKLearnSpec{
					PredictorExtensionSpec: kfservingv1.PredictorExtensionSpec{
						StorageURI:      &infSvc.Spec.ModelUri,
						ProtocolVersion: &defaultProtocol,
						RuntimeVersion:  &runtimeVersion,
						Container: v1.Container{
							Name: "kfserving-container",
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"cpu":    resource.MustParse("1000m"),
									"memory": resource.MustParse("2Gi"),
								},
								Requests: v1.ResourceList{
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("128Mi"),
								},
							},
						},
					},
				},
			},
		}

		kfsvcr := kfserving.NewKfservingReconciler(r.Client, r.Scheme, objectMeta, &spec)

		if err := controllerutil.SetControllerReference(infSvc, kfsvcr.Service, r.Scheme); err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "fails to set owner reference for predictor")
		}

		status, err := kfsvcr.Reconcile()
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "fails to reconcile kfserving inference service")
		}

		infSvc.Status.PropagateStatusFromKfserving(status)
	} else if infSvc.Spec.Backend == "seldon" {
		replicas := int32(1)
		impl := seldonv1.PredictiveUnitImplementation(seldonv1const.PrePackedServerSklearn)
		spec := seldonv1.SeldonDeploymentSpec{
			Name: infSvc.Name,
			Predictors: []seldonv1.PredictorSpec{{
				Name:     infSvc.Name,
				Replicas: &replicas,
				Graph: seldonv1.PredictiveUnit{
					Implementation:   &impl,
					ModelURI:         infSvc.Spec.ModelUri,
					Name:             "classifier",
					EnvSecretRefName: infSvc.Spec.ServiceAccountName,
					Parameters: []seldonv1.Parameter{{
						Name:  "method",
						Type:  seldonv1.STRING,
						Value: "predict",
					}},
				}},
			},
		}

		seldonr := seldon.NewSeldonReconciler(r.Client, r.Scheme, objectMeta, &spec)

		if err := controllerutil.SetControllerReference(infSvc, seldonr.Service, r.Scheme); err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "fails to set owner reference for predictor")
		}

		status, err := seldonr.Reconcile()
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "fails to reconcile seldon inference service")
		}

		infSvc.Status.PropagateStatusFromSeldon(status)
	}

	if err := r.updateStatus(infSvc); err != nil {
		r.Recorder.Eventf(infSvc, v1.EventTypeWarning, "InternalError", err.Error())
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *InferenceServiceReconciler) updateStatus(desiredService *servingv1.InferenceService) error {
	existingService := &servingv1.InferenceService{}
	namespacedName := types.NamespacedName{Name: desiredService.Name, Namespace: desiredService.Namespace}
	if err := r.Get(context.TODO(), namespacedName, existingService); err != nil {
		return err
	}

	wasAvailable := isInferenceServiceAvailable(existingService.Status)
	if equality.Semantic.DeepEqual(existingService.Status, desiredService.Status) {
		// If we didn't change anything then don't call updateStatus.
		// This is important because the copy we loaded from the informer's
		// cache may be stale and we don't want to overwrite a prior update
		// to status with this stale state.
	} else if err := r.Status().Update(context.TODO(), desiredService); err != nil {
		// It might happen that the reconciler tries to update an object that did not finish
		// updating yet, in that case just ignore the error
		switch se := err.(type) {
		case *apierr.StatusError:
			if se.Status().Reason == metav1.StatusReasonConflict {
				return nil
			}
		}
		r.Log.Error(err, "Failed to update InferenceService status", "InferenceService", desiredService.Name)
		r.Recorder.Eventf(desiredService, v1.EventTypeWarning, "UpdateFailed",
			"Failed to update status for InferenceService %q: %v", desiredService.Name, err)
		return errors.Wrapf(err, "fails to update InferenceService status")
	} else {
		// If there was a difference and there was no error.
		isAvailable := isInferenceServiceAvailable(desiredService.Status)
		if wasAvailable && !isAvailable { // Moved to a different State
			r.Recorder.Eventf(desiredService, v1.EventTypeWarning, string(servingv1.StatusStateCreating),
				fmt.Sprintf("InferenceService [%v] is no longer Available", desiredService.GetName()))
		} else if !wasAvailable && isAvailable { // Moved to Available State
			r.Recorder.Eventf(desiredService, v1.EventTypeNormal, string(servingv1.StatusStateAvailable),
				fmt.Sprintf("InferenceService [%v] is Available", desiredService.GetName()))
		}
	}
	return nil
}

func isInferenceServiceAvailable(status servingv1.InferenceServiceStatus) bool {
	return status.Status == servingv1.StatusStateAvailable
}

func (r *InferenceServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&servingv1.InferenceService{}).
		Owns(&kfservingv1.InferenceService{}).
		Owns(&seldonv1.SeldonDeployment{}).
		Complete(r)
}
