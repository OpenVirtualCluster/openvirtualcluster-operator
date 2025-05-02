/*
Copyright 2025.

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
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VirtualClusterSpec defines the desired state of VirtualCluster.
type VirtualClusterSpec struct {
	Chart HelmChart `json:"chart,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Values *apiextensionsv1.JSON `json:"values,required"`
}

// GetValues unmarshals the raw values to a map[string]interface{} and returns
// the result.
func (in VirtualCluster) GetValues() (map[string]interface{}, error) {
	var values map[string]interface{}
	if in.Spec.Values != nil {
		// For raw JSON values
		if err := json.Unmarshal(in.Spec.Values.Raw, &values); err != nil {
			// If raw JSON unmarshal fails, try YAML unmarshal
			if yamlErr := yaml.Unmarshal(in.Spec.Values.Raw, &values); yamlErr != nil {
				return nil, fmt.Errorf("failed to unmarshal values as JSON or YAML: %v, %v", err, yamlErr)
			}
		}
	}
	// Initialize an empty map if values is nil
	if values == nil {
		values = make(map[string]interface{})
	}
	return values, nil
}

type HelmChart struct {
	// Version is the version of the helm chart
	// +default:value="v0.24.1"
	Version string `json:"version,omitempty"`
}

// VirtualClusterStatus defines the observed state of VirtualCluster.
type VirtualClusterStatus struct {
	// Phase is the current phase of the VirtualCluster
	// +optional
	Phase VirtualClusterPhase `json:"phase,omitempty"`

	// Conditions represent the latest available observations of an object's state
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// Message provides human-readable details about the current status
	// +optional
	Message string `json:"message,omitempty"`

	// HelmChart is the name of the helm chart used to deploy the VirtualCluster
	// +optional
	HelmChart string `json:"helmChart,omitempty"`

	// HelmRelease is the name of the helm release used to deploy the VirtualCluster
	// +optional
	HelmRelease string `json:"helmRelease,omitempty"`
}

// VirtualClusterPhase is a label for the phase of a VirtualCluster at the current time.
type VirtualClusterPhase string

// These are the valid phases of a VirtualCluster.
const (
	// VirtualClusterPending means the VirtualCluster has been created/added to the system, but is not yet being processed.
	VirtualClusterPending VirtualClusterPhase = "Pending"

	// VirtualClusterProvisioning means the VirtualCluster is being deployed.
	VirtualClusterProvisioning VirtualClusterPhase = "Provisioning"

	// VirtualClusterRunning means the VirtualCluster has been deployed successfully.
	VirtualClusterRunning VirtualClusterPhase = "Running"

	// VirtualClusterFailed means the VirtualCluster failed to be deployed or is in an error state.
	VirtualClusterFailed VirtualClusterPhase = "Failed"

	// VirtualClusterDeleting means the VirtualCluster is being deleted.
	VirtualClusterDeleting VirtualClusterPhase = "Deleting"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="Status of the VirtualCluster"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:shortName=vc
// +kubebuilder:storageversion

// VirtualCluster is the Schema for the virtualclusters API.
type VirtualCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualClusterSpec   `json:"spec,omitempty"`
	Status VirtualClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VirtualClusterList contains a list of VirtualCluster.
type VirtualClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualCluster{}, &VirtualClusterList{})
}
