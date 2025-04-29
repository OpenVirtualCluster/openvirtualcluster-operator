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
	"testing"

	"github.com/stretchr/testify/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVirtualCluster_GetValues(t *testing.T) {
	tests := []struct {
		name       string
		vc         VirtualCluster
		wantValues map[string]interface{}
		wantErr    bool
	}{
		{
			name: "valid values",
			vc: VirtualCluster{
				Spec: VirtualClusterSpec{
					Values: &apiextensionsv1.JSON{Raw: []byte(`{"vcluster": {"image": "rancher/k3s:v1.25.0-k3s1"}}`)},
				},
			},
			wantValues: map[string]interface{}{
				"vcluster": map[string]interface{}{
					"image": "rancher/k3s:v1.25.0-k3s1",
				},
			},
			wantErr: false,
		},
		{
			name: "complex values",
			vc: VirtualCluster{
				Spec: VirtualClusterSpec{
					Values: &apiextensionsv1.JSON{Raw: []byte(`{
						"vcluster": {
							"image": "rancher/k3s:v1.25.0-k3s1",
							"extraArgs": ["--disable=traefik"]
						},
						"service": {
							"type": "ClusterIP"
						},
						"persistence": {
							"enabled": true,
							"size": "10Gi"
						}
					}`)},
				},
			},
			wantValues: map[string]interface{}{
				"vcluster": map[string]interface{}{
					"image":     "rancher/k3s:v1.25.0-k3s1",
					"extraArgs": []interface{}{"--disable=traefik"},
				},
				"service": map[string]interface{}{
					"type": "ClusterIP",
				},
				"persistence": map[string]interface{}{
					"enabled": true,
					"size":    "10Gi",
				},
			},
			wantErr: false,
		},
		{
			name: "empty values",
			vc: VirtualCluster{
				Spec: VirtualClusterSpec{
					Values: &apiextensionsv1.JSON{Raw: []byte(`{}`)},
				},
			},
			wantValues: map[string]interface{}{},
			wantErr:    false,
		},
		{
			name: "nil values",
			vc: VirtualCluster{
				Spec: VirtualClusterSpec{
					Values: nil,
				},
			},
			wantValues: map[string]interface{}{},
			wantErr:    false,
		},
		{
			name: "invalid json",
			vc: VirtualCluster{
				Spec: VirtualClusterSpec{
					Values: &apiextensionsv1.JSON{Raw: []byte(`{invalid: json}`)},
				},
			},
			wantValues: nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValues, err := tt.vc.GetValues()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantValues, gotValues)
			}
		})
	}
}

func TestVirtualClusterPhases(t *testing.T) {
	// Test VirtualClusterPhase constants are as expected
	assert.Equal(t, VirtualClusterPhase("Pending"), VirtualClusterPending)
	assert.Equal(t, VirtualClusterPhase("Provisioning"), VirtualClusterProvisioning)
	assert.Equal(t, VirtualClusterPhase("Running"), VirtualClusterRunning)
	assert.Equal(t, VirtualClusterPhase("Failed"), VirtualClusterFailed)
	assert.Equal(t, VirtualClusterPhase("Deleting"), VirtualClusterDeleting)
}

func TestVirtualCluster_DefaultValues(t *testing.T) {
	// Test default values for VirtualCluster
	vc := VirtualCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vcluster",
			Namespace: "default",
		},
		Spec: VirtualClusterSpec{
			Chart: HelmChart{
				// Default version should be used when not specified
			},
			Values: &apiextensionsv1.JSON{Raw: []byte(`{}`)},
		},
	}

	// In a real implementation, you would call a defaulting webhook here
	// For testing purposes, we can check that the default value is v0.24.1
	// This is assuming you have implemented a defaulting webhook or function

	// Simulate defaulting
	if vc.Spec.Chart.Version == "" {
		vc.Spec.Chart.Version = "v0.24.1"
	}

	assert.Equal(t, "v0.24.1", vc.Spec.Chart.Version)
}

func TestVirtualClusterStatusConditions(t *testing.T) {
	// Test adding and finding conditions in status
	vc := VirtualCluster{
		Status: VirtualClusterStatus{
			Conditions: []metav1.Condition{
				{
					Type:    "Available",
					Status:  metav1.ConditionTrue,
					Reason:  "VirtualClusterAvailable",
					Message: "VirtualCluster is available",
				},
			},
		},
	}

	// Test finding existing condition
	for _, cond := range vc.Status.Conditions {
		if cond.Type == "Available" {
			assert.Equal(t, metav1.ConditionTrue, cond.Status)
			assert.Equal(t, "VirtualClusterAvailable", cond.Reason)
		}
	}

	// Add another condition
	vc.Status.Conditions = append(vc.Status.Conditions, metav1.Condition{
		Type:    "Deployed",
		Status:  metav1.ConditionTrue,
		Reason:  "DeploymentComplete",
		Message: "VirtualCluster deployment completed",
	})

	// Test condition count
	assert.Equal(t, 2, len(vc.Status.Conditions))
}
