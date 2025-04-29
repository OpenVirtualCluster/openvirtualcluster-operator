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

package controller

import (
	"context"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	controllerruntime "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1alpha1 "github.com/prakashmishra1598/openvc/api/v1alpha1"
)

// TestReconcilerSetup creates a test reconciler with the necessary dependencies
func TestReconcilerSetup() (*VirtualClusterReconciler, client.Client, *runtime.Scheme) {
	// Create a scheme
	s := runtime.NewScheme()
	_ = corev1alpha1.AddToScheme(s)
	_ = scheme.AddToScheme(s)

	// Create fake client
	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()

	// Create the reconciler
	reconciler := &VirtualClusterReconciler{
		Client:   fakeClient,
		Scheme:   s,
		Recorder: record.NewFakeRecorder(10),
	}

	return reconciler, fakeClient, s
}

// CreateTestVirtualCluster creates a test VirtualCluster instance
func CreateTestVirtualCluster(name, namespace string, values string) *corev1alpha1.VirtualCluster {
	if values == "" {
		values = `{"vcluster": {"image": "rancher/k3s:v1.25.0-k3s1"}}`
	}

	return &corev1alpha1.VirtualCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1alpha1.VirtualClusterSpec{
			Chart: corev1alpha1.HelmChart{
				Version: "v0.24.1",
			},
			Values: &apiextensionsv1.JSON{Raw: []byte(values)},
		},
	}
}

// ReconcileAndUpdateVirtualCluster performs a reconciliation and returns the updated VirtualCluster
func ReconcileAndUpdateVirtualCluster(ctx context.Context, reconciler *VirtualClusterReconciler, vc *corev1alpha1.VirtualCluster) (*corev1alpha1.VirtualCluster, reconcile.Result, error) {
	// Create a NamespacedName for the request
	namespacedName := types.NamespacedName{
		Name:      vc.Name,
		Namespace: vc.Namespace,
	}

	// Reconcile
	result, err := reconciler.Reconcile(ctx, reconcile.Request{
		NamespacedName: namespacedName,
	})

	// Get the updated VirtualCluster
	updatedVC := &corev1alpha1.VirtualCluster{}
	getErr := reconciler.Client.Get(ctx, namespacedName, updatedVC)
	if getErr != nil {
		return nil, result, getErr
	}

	return updatedVC, result, err
}

// SetupTestReconcileLoop performs multiple reconciliations to simulate the controller loop
func SetupTestReconcileLoop(ctx context.Context, reconciler *VirtualClusterReconciler, vc *corev1alpha1.VirtualCluster, iterations int) (*corev1alpha1.VirtualCluster, error) {
	var err error
	updatedVC := vc

	// Create the VirtualCluster
	err = reconciler.Client.Create(ctx, vc)
	if err != nil {
		return nil, err
	}

	// Run multiple reconciliations
	for i := 0; i < iterations; i++ {
		updatedVC, _, err = ReconcileAndUpdateVirtualCluster(ctx, reconciler, updatedVC)
		if err != nil {
			return updatedVC, err
		}

		// Short sleep to simulate reconcile loop timing
		time.Sleep(10 * time.Millisecond)
	}

	return updatedVC, nil
}

// CustomizableControllerOptions returns controller options that can be customized for testing
func CustomizableControllerOptions() controllerruntime.Options {
	recoverPanic := true
	return controllerruntime.Options{
		MaxConcurrentReconciles: 1,
		RecoverPanic:            &recoverPanic,
	}
}

// MarkVirtualClusterForDeletion marks a VirtualCluster for deletion
func MarkVirtualClusterForDeletion(ctx context.Context, client client.Client, vc *corev1alpha1.VirtualCluster) error {
	// Add finalizer if it doesn't exist
	if !containsString(vc.Finalizers, vclusterFinalizer) {
		vc.Finalizers = append(vc.Finalizers, vclusterFinalizer)
		if err := client.Update(ctx, vc); err != nil {
			return err
		}
	}

	// Mark for deletion
	now := metav1.Now()
	vc.DeletionTimestamp = &now
	return client.Update(ctx, vc)
}

// Helper function to check if a string exists in a slice
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
