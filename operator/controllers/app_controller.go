/*
Copyright 2020 Codelogia

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
	"reflect"
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	manorv1 "github.com/codelogia/manor/operator/api/v1"
)

// AppReconciler reconciles an App object.
type AppReconciler struct {
	client.Client
	Log                  logr.Logger
	Scheme               *runtime.Scheme
	DefaultImageRegistry string
}

// SetupAppReconciler sets up the App reconciler.
func SetupAppReconciler(mgr ctrl.Manager, defaultImageRegistry string) error {
	r := &AppReconciler{
		Client:               mgr.GetClient(),
		Log:                  ctrl.Log.WithName("controllers").WithName("App"),
		Scheme:               mgr.GetScheme(),
		DefaultImageRegistry: defaultImageRegistry,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&manorv1.App{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=manor.codelogia.com,resources=apps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=manor.codelogia.com,resources=apps/status,verbs=get;update;patch

// Reconcile reconciles the App resources.
func (r *AppReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	defer cancel()

	log := r.Log.WithValues("app", req.NamespacedName)

	app := &manorv1.App{}
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		if errors.IsNotFound(err) {
			log.Info("App resource deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	var imageRegistry string
	if app.Spec.ImageRegistry != "" {
		imageRegistry = app.Spec.ImageRegistry
	} else {
		imageRegistry = r.DefaultImageRegistry
	}

	if strings.HasSuffix(imageRegistry, ".svc") {
		split := strings.Split(imageRegistry, ".")
		if len(split) != 3 {
			err := fmt.Errorf("image registry %q is not in the format <name>.<namespace>.svc", imageRegistry)
			return ctrl.Result{Requeue: false}, err
		}
		imageRegistryNamespacedName := types.NamespacedName{
			Name:      split[0],
			Namespace: split[1],
		}
		imageRegistryService := &corev1.Service{}
		if err := r.Get(ctx, imageRegistryNamespacedName, imageRegistryService); err != nil {
			if errors.IsNotFound(err) {
				log.Info("Image registry not found, retrying...")
				return ctrl.Result{RequeueAfter: time.Second * 3}, nil
			}
			return ctrl.Result{}, err
		}
		var nodePort uint16
		for _, port := range imageRegistryService.Spec.Ports {
			if port.NodePort != 0 {
				nodePort = uint16(port.NodePort)
			}
		}
		imageRegistry = fmt.Sprintf("127.0.0.1:%d", nodePort)
	}

	imagePullPolicy := app.Spec.ImagePullPolicy
	if imagePullPolicy == "" {
		imagePullPolicy = corev1.PullIfNotPresent
	}

	replicas := app.Spec.Replicas
	if replicas == nil {
		replicas = new(int32)
		*replicas = 1
	}

	resources := app.Spec.Resources
	if resources == nil {
		resources = &corev1.ResourceRequirements{}
	}

	var command []string
	if app.Spec.Entrypoint != "" {
		command = []string{app.Spec.Entrypoint}
	}

	var args []string
	if len(app.Spec.Args) > 0 {
		args = app.Spec.Args
	}

	labels := map[string]string{"manor.codelogia.com/app": app.Name}

	desiredAppContainerPort := corev1.ContainerPort{
		Name:          "http",
		Protocol:      corev1.ProtocolTCP,
		ContainerPort: 8080,
	}

	desiredDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:           fmt.Sprintf("%s/%s/%s", imageRegistry, app.Namespace, app.Name),
						ImagePullPolicy: imagePullPolicy,
						Name:            app.Name,
						Command:         command,
						Args:            args,
						// TODO(f0rmiga): remove this and add a sidecar for mTLS with the router.
						Ports: []corev1.ContainerPort{desiredAppContainerPort},
						Env: []corev1.EnvVar{
							{
								Name:  "PORT",
								Value: fmt.Sprintf("%d", desiredAppContainerPort.ContainerPort),
							},
						},
					}},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(app, desiredDeployment, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	currentDeployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: desiredDeployment.Name, Namespace: desiredDeployment.Namespace}, currentDeployment); err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		log.Info(
			"Creating Deployment",
			"Deployment.Namespace", desiredDeployment.Namespace,
			"Deployment.Name", desiredDeployment.Name,
		)

		if err := r.Create(ctx, desiredDeployment); err != nil {
			log.Error(
				err, "Failed to create Deployment",
				"Deployment.Namespace", desiredDeployment.Namespace,
				"Deployment.Name", desiredDeployment.Name,
			)
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	if message, needsUpdate := r.deploymentNeedsUpdate(desiredDeployment, currentDeployment); needsUpdate {
		log.Info(
			"Updating Deployment",
			"message", message,
			"Deployment.Namespace", desiredDeployment.Namespace,
			"Deployment.Name", desiredDeployment.Name,
		)
		if err := r.Update(ctx, desiredDeployment); err != nil {
			log.Error(
				err, "Failed to update Deployment",
				"Deployment.Namespace", desiredDeployment.Namespace,
				"Deployment.Name", desiredDeployment.Name,
			)
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	desiredService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{{
				Name:       desiredAppContainerPort.Name,
				Protocol:   desiredAppContainerPort.Protocol,
				Port:       desiredAppContainerPort.ContainerPort,
				TargetPort: intstr.FromInt(int(desiredAppContainerPort.ContainerPort)),
			}},
			Selector: labels,
		},
	}

	if err := ctrl.SetControllerReference(app, desiredService, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	currentService := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: desiredService.Name, Namespace: desiredService.Namespace}, currentService); err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		log.Info(
			"Creating Service",
			"Service.Namespace", desiredService.Namespace,
			"Service.Name", desiredService.Name,
		)

		if err := r.Create(ctx, desiredService); err != nil {
			log.Error(
				err, "Failed to create Service",
				"Service.Namespace", desiredService.Namespace,
				"Service.Name", desiredService.Name,
			)
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	if message, needsRecreate := r.serviceNeedsRecreate(desiredService, currentService); needsRecreate {
		log.Info(
			"Recreating Service",
			"message", message,
			"Service.Namespace", desiredService.Namespace,
			"Service.Name", desiredService.Name,
		)

		if err := r.Delete(ctx, desiredService); err != nil {
			log.Error(
				err, "Failed to delete Service",
				"Service.Namespace", desiredService.Namespace,
				"Service.Name", desiredService.Name,
			)
			return ctrl.Result{}, err
		}

		// If the service fails to recreate with the same IP, the service will get a new ClusterIP
		// once the reconciler notices a missing service in the next reconcile loop.
		desiredService.Spec.ClusterIP = currentService.Spec.ClusterIP
		if err := r.Create(ctx, desiredService); err != nil {
			log.Error(
				err, "Failed to recreate Service",
				"Service.Namespace", desiredService.Namespace,
				"Service.Name", desiredService.Name,
			)
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	if message, needsUpdate := r.serviceNeedsUpdate(desiredService, currentService); needsUpdate {
		log.Info(
			"Updating Service",
			"message", message,
			"Service.Namespace", desiredService.Namespace,
			"Service.Name", desiredService.Name,
		)
		if err := r.Update(ctx, desiredService); err != nil {
			log.Error(
				err, "Failed to update Service",
				"Service.Namespace", desiredService.Namespace,
				"Service.Name", desiredService.Name,
			)
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{RequeueAfter: time.Second * 15}, nil
}

func (r *AppReconciler) deploymentNeedsUpdate(desired, current *appsv1.Deployment) (string, bool) {
	var desiredReplicas, currentReplicas int32
	if desired.Spec.Replicas != nil {
		desiredReplicas = *desired.Spec.Replicas
	}
	if current.Spec.Replicas != nil {
		currentReplicas = *current.Spec.Replicas
	}
	if desiredReplicas != currentReplicas {
		return fmt.Sprintf(
			"current number of replicas %d doesn't match desired %d",
			currentReplicas, desiredReplicas,
		), true
	}

	if len(desired.Spec.Template.Spec.Containers) != len(current.Spec.Template.Spec.Containers) {
		return fmt.Sprintf(
			"current containers size %d doesn't match desired %d",
			len(current.Spec.Template.Spec.Containers), len(desired.Spec.Template.Spec.Containers),
		), true
	}

	desiredAppContainer := desired.Spec.Template.Spec.Containers[0]
	currentAppContainer := current.Spec.Template.Spec.Containers[0]
	if desiredAppContainer.Image != currentAppContainer.Image {
		return fmt.Sprintf(
			"current container image %s doesn't match desired %s",
			currentAppContainer.Image, desiredAppContainer.Image,
		), true
	}

	if desiredAppContainer.ImagePullPolicy != currentAppContainer.ImagePullPolicy {
		return fmt.Sprintf(
			"current container imagePullPolicy %s doesn't match desired %s",
			currentAppContainer.ImagePullPolicy, desiredAppContainer.ImagePullPolicy,
		), true
	}

	if !equalStringSlice(desiredAppContainer.Command, currentAppContainer.Command) {
		return fmt.Sprintf(
			"current container command %v doesn't match desired %v",
			currentAppContainer.Command, desiredAppContainer.Command,
		), true
	}

	if !equalStringSlice(desiredAppContainer.Args, currentAppContainer.Args) {
		return fmt.Sprintf(
			"current container args %v doesn't match desired %v",
			currentAppContainer.Args, desiredAppContainer.Args,
		), true
	}

	return "", false
}

func (r *AppReconciler) serviceNeedsRecreate(desired, current *corev1.Service) (string, bool) {
	if !reflect.DeepEqual(desired.Spec.Selector, current.Spec.Selector) {
		return fmt.Sprintf(
			"current selector %v doesn't match desired %v",
			current.Spec.Selector, desired.Spec.Selector,
		), true
	}

	return "", false
}

func (r *AppReconciler) serviceNeedsUpdate(desired, current *corev1.Service) (string, bool) {
	if len(desired.Spec.Ports) != len(current.Spec.Ports) {
		return fmt.Sprintf(
			"current number of ports %d doesn't match desired %d",
			len(current.Spec.Ports), len(desired.Spec.Ports),
		), true
	}

	desiredPort := desired.Spec.Ports[0]
	currentPort := current.Spec.Ports[0]

	if desiredPort.Name != currentPort.Name {
		return fmt.Sprintf(
			"current port name %s doesn't match desired %s",
			currentPort.Name, desiredPort.Name,
		), true
	}

	if desiredPort.Protocol != currentPort.Protocol {
		return fmt.Sprintf(
			"current port protocol %s doesn't match desired %s",
			currentPort.Protocol, desiredPort.Protocol,
		), true
	}

	if desiredPort.Port != currentPort.Port {
		return fmt.Sprintf(
			"current port number %d doesn't match desired %d",
			currentPort.Port, desiredPort.Port,
		), true
	}

	if desiredPort.TargetPort.IntVal != currentPort.TargetPort.IntVal {
		return fmt.Sprintf(
			"current port target number %d doesn't match desired %d",
			currentPort.TargetPort.IntVal, desiredPort.TargetPort.IntVal,
		), true
	}

	return "", false
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}
