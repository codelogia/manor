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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AppSpec defines the desired state of App.
type AppSpec struct {
	// The image registry to override the default Image Registry.
	ImageRegistry string `json:"imageRegistry,omitempty"`
	// Image pull policy.
	// One of Always, Never, IfNotPresent.
	// Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// The number of replicas for the App.
	Replicas *int32 `json:"replicas,omitempty"`
	// Compute Resources required by the App.
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// The entrypoint command for the App.
	Entrypoint string `json:"entrypoint,omitempty"`
	// The arguments for the entrypoint command of the App.
	Args []string `json:"args,omitempty"`
}

// AppStatus defines the observed state of App.
type AppStatus struct {
	// Current service state of App.
	Conditions []AppCondition `json:"conditions,omitempty"`
}

// AppCondition represents App conditions.
type AppCondition struct {
	// Type is the type of the condition.
	Type AppConditionType `json:"type"`
	// Status is the status of the condition.
	// Can be True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
}

// AppConditionType represents App condition types.
type AppConditionType string

const (
	// AppInitialized means that all replicas have been initialized but are not running yet.
	AppInitialized AppConditionType = "Initialized"
	// AppReady means the App is able to handle requests.
	AppReady AppConditionType = "Ready"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// App is the Schema for the apps API.
type App struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppSpec   `json:"spec,omitempty"`
	Status AppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AppList contains a list of App.
type AppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []App `json:"items"`
}

func init() {
	SchemeBuilder.Register(&App{}, &AppList{})
}
