/*
Copyright 2021.

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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// StarterKitSpec defines the desired state of StarterKit
type StarterKitSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Options      StarterKitSpecOptions  `json:"options,omitempty"`
	TemplateRepo StarterKitSpecTemplate `json:"templateRepo"`
}

type StarterKitSpecOptions struct {
	Port int32           `json:"port"`
	Env  []corev1.EnvVar `json:"env"`
}

type StarterKitSpecTemplate struct {
	TemplateOwner    string                   `json:"templateOwner"`
	TemplateRepoName string                   `json:"templateRepoName"`
	Owner            string                   `json:"owner"`
	Name             string                   `json:"name"`
	Description      string                   `json:"repoDescription"`
	SecretKeyRef     corev1.SecretKeySelector `json:"secretKeyRef"`
}

// StarterKitStatus defines the observed state of StarterKit
type StarterKitStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	TargetRepo string `json:"targetRepo"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// StarterKit is the Schema for the starterkits API
type StarterKit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StarterKitSpec   `json:"spec,omitempty"`
	Status StarterKitStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StarterKitList contains a list of StarterKit
type StarterKitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StarterKit `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StarterKit{}, &StarterKitList{})
}
