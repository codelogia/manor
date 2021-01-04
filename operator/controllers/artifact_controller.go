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
	"crypto/rand"
	"fmt"
	"time"

	"github.com/go-logr/logr"
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

// ArtifactReconciler reconciles a Artifact object.
type ArtifactReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	DockerHost           string
	DefaultImageRegistry string
	AppBuilderImage      string
}

// SetupArtifactReconciler sets up the Artifact reconciler.
func SetupArtifactReconciler(
	mgr ctrl.Manager,
	dockerHost string,
	defaultImageRegistry string,
	appBuilderImage string,
) error {
	r := &ArtifactReconciler{
		Client:               mgr.GetClient(),
		Log:                  ctrl.Log.WithName("controllers").WithName("Artifact"),
		Scheme:               mgr.GetScheme(),
		DockerHost:           dockerHost,
		DefaultImageRegistry: defaultImageRegistry,
		AppBuilderImage:      appBuilderImage,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&manorv1.Artifact{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=manor.codelogia.com,resources=artifacts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=manor.codelogia.com,resources=artifacts/status,verbs=get;update;patch

// Reconcile reconciles the Artifact resources.
func (r *ArtifactReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	defer cancel()

	log := r.Log.WithValues("artifact", req.NamespacedName)

	artifact := &manorv1.Artifact{}
	if err := r.Get(ctx, req.NamespacedName, artifact); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Artifact resource deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if artifact.Spec.App == "" {
		err := fmt.Errorf("spec.App cannot be empty, not requeueing")
		return ctrl.Result{Requeue: false}, err
	}

	if len(artifact.Status.Conditions) == 0 {
		condition := manorv1.ArtifactCondition{
			Type:   manorv1.ArtifactInitialized,
			Status: corev1.ConditionTrue,
		}
		artifact.Status.Conditions = []manorv1.ArtifactCondition{condition}
		if err := r.Status().Update(ctx, artifact); err != nil {
			log.Error(
				err, "Failed to update Artifact status",
				"Artifact.Namespace", artifact.Namespace,
				"Artifact.Name", artifact.Name,
			)
			return ctrl.Result{}, err
		}
		// Do not requeue as the artifact update will trigger another event.
		return ctrl.Result{}, nil
	}

	labels := map[string]string{"manor.codelogia.com/app": artifact.Spec.App}

	secretName := fmt.Sprintf("%s-app-builder-creds", artifact.Spec.App)

	currentSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: artifact.Namespace}, currentSecret); err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		log.Info(
			"Creating Secret with artifact credentials",
			"Secret.Namespace", artifact.Namespace,
			"Secret.Name", secretName,
		)

		tokenBytes := make([]byte, 32)
		_, err := rand.Read(tokenBytes)
		if err != nil {
			log.Error(
				err, "Failed to create Secret with artifact credentials",
				"Secret.Namespace", artifact.Namespace,
				"Secret.Name", secretName,
			)
			return ctrl.Result{}, err
		}
		token := fmt.Sprintf("%x", tokenBytes)

		desiredSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: artifact.Namespace,
				Labels:    labels,
			},
			Data: map[string][]byte{
				"token": []byte(token),
			},
		}

		if err := ctrl.SetControllerReference(artifact, desiredSecret, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, desiredSecret); err != nil {
			log.Error(
				err, "Failed to create Secret with artifact credentials",
				"Secret.Namespace", artifact.Namespace,
				"Secret.Name", secretName,
			)
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	podName := fmt.Sprintf("%s-app-builder", artifact.Spec.App)
	podAddrPort := 8081

	var imageRegistry string
	if artifact.Spec.ImageRegistry != "" {
		imageRegistry = artifact.Spec.ImageRegistry
	} else {
		imageRegistry = r.DefaultImageRegistry
	}

	desiredPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: artifact.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{{
				Name:            "app-builder",
				Image:           r.AppBuilderImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				SecurityContext: &corev1.SecurityContext{
					RunAsUser:                func(v int64) *int64 { return &v }(1000),
					RunAsNonRoot:             func(v bool) *bool { return &v }(true),
					AllowPrivilegeEscalation: func(v bool) *bool { return &v }(false),
					ReadOnlyRootFilesystem:   func(v bool) *bool { return &v }(true),
				},
				Ports: []corev1.ContainerPort{corev1.ContainerPort{
					Name:          "http",
					Protocol:      corev1.ProtocolTCP,
					ContainerPort: 8081,
				}},
				Env: []corev1.EnvVar{
					{
						Name:  "DOCKER_HOST",
						Value: r.DockerHost,
					},
					{
						Name:  "ADDR",
						Value: fmt.Sprintf(":%d", podAddrPort),
					},
					{
						Name:  "BUILD_DIR",
						Value: "/tmp/build",
					},
					{
						Name: "TOKEN",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: secretName,
								},
								Key: "token",
							},
						},
					},
					{
						Name:  "APP_NAMESPACE",
						Value: artifact.Namespace,
					},
					{
						Name:  "APP_NAME",
						Value: artifact.Spec.App,
					},
					{
						Name:  "IMAGE_REGISTRY",
						Value: imageRegistry,
					},
				},
				ReadinessProbe: &corev1.Probe{
					Handler: corev1.Handler{
						TCPSocket: &corev1.TCPSocketAction{
							Port: intstr.FromInt(podAddrPort),
						},
					},
					InitialDelaySeconds: 3,
					PeriodSeconds:       3,
				},
				LivenessProbe: &corev1.Probe{
					Handler: corev1.Handler{
						TCPSocket: &corev1.TCPSocketAction{
							Port: intstr.FromInt(podAddrPort),
						},
					},
					InitialDelaySeconds: 15,
					PeriodSeconds:       10,
				},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "tmp",
					ReadOnly:  false,
					MountPath: "/tmp",
				}},
			}},
			Volumes: []corev1.Volume{{
				Name:         "tmp",
				VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
			}},
		},
	}

	if err := ctrl.SetControllerReference(artifact, desiredPod, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	currentPod := &corev1.Pod{}
	if err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: artifact.Namespace}, currentPod); err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		log.Info(
			"Creating Pod for building app artifact",
			"Pod.Namespace", desiredPod.Namespace,
			"Pod.Name", desiredPod.Name,
		)

		if err := r.Create(ctx, desiredPod); err != nil {
			log.Error(
				err, "Failed to create Pod for building app artifact",
				"Pod.Namespace", desiredPod.Namespace,
				"Pod.Name", desiredPod.Name,
			)
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	// If the pod is completed (succeeded or failed), the artifact build should also be marked as completed.
	if currentPod.Status.Phase == corev1.PodSucceeded || currentPod.Status.Phase == corev1.PodFailed {
		buildCompleted := false
		for _, condition := range artifact.Status.Conditions {
			if condition.Type == manorv1.ArtifactCompleted && condition.Status == corev1.ConditionTrue {
				buildCompleted = true
				break
			}
		}
		if !buildCompleted {
			condition := manorv1.ArtifactCondition{
				Type:   manorv1.ArtifactCompleted,
				Status: corev1.ConditionTrue,
			}
			artifact.Status.Conditions = append(artifact.Status.Conditions, condition)
			if err := r.Status().Update(ctx, artifact); err != nil {
				log.Error(
					err, "Failed to update Artifact status",
					"Artifact.Namespace", artifact.Namespace,
					"Artifact.Name", artifact.Name,
				)
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	podReady := false
	for _, condition := range currentPod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			podReady = true
			break
		}
	}
	if !podReady {
		return ctrl.Result{Requeue: true}, nil
	}

	buildInProgress := false
	for _, condition := range artifact.Status.Conditions {
		if condition.Type == manorv1.ArtifactInProgress && condition.Status == corev1.ConditionTrue {
			buildInProgress = true
			break
		}
	}
	if !buildInProgress {
		condition := manorv1.ArtifactCondition{
			Type:   manorv1.ArtifactInProgress,
			Status: corev1.ConditionTrue,
		}
		artifact.Status.Conditions = append(artifact.Status.Conditions, condition)
		if err := r.Status().Update(ctx, artifact); err != nil {
			log.Error(
				err, "Failed to update Artifact status",
				"Artifact.Namespace", artifact.Namespace,
				"Artifact.Name", artifact.Name,
			)
			return ctrl.Result{}, err
		}
		// Do not requeue as the artifact update will trigger another event.
		return ctrl.Result{}, nil
	}
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}
