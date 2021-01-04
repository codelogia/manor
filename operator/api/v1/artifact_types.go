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

// ArtifactSpec defines the desired state of Artifact.
type ArtifactSpec struct {
	// The name of the App the artifact is tied to.
	App string `json:"app,omitempty"`
	// The image registry to override the default Image Registry.
	ImageRegistry string `json:"imageRegistry,omitempty"`
}

// ArtifactStatus defines the observed state of Artifact.
type ArtifactStatus struct {
	// Current service state of Artifact.
	Conditions []ArtifactCondition `json:"conditions,omitempty"`
}

// ArtifactCondition represents Artifact conditions.
type ArtifactCondition struct {
	// Type is the type of the condition.
	Type ArtifactConditionType `json:"type"`
	// Status is the status of the condition.
	// Can be True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
}

// ArtifactConditionType represents Artifact condition types.
type ArtifactConditionType string

const (
	// ArtifactInitialized means that the Artifact was initialized but is not ready yet.
	ArtifactInitialized ArtifactConditionType = "Initialized"
	// ArtifactInProgress means that the Artifact is in progress.
	ArtifactInProgress ArtifactConditionType = "In progress"
	// ArtifactCompleted means the Artifact is completed.
	ArtifactCompleted ArtifactConditionType = "Completed"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Artifact is the Schema for the artifacts API.
type Artifact struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArtifactSpec   `json:"spec,omitempty"`
	Status ArtifactStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ArtifactList contains a list of Artifact.
type ArtifactList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Artifact `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Artifact{}, &ArtifactList{})
}
