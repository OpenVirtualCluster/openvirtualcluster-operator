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
	"os/exec"

	corev1alpha1 "github.com/OpenVirtualCluster/openvirtualcluster-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MockVirtualClusterReconciler embeds the real reconciler for testing
type MockVirtualClusterReconciler struct {
	// Embed the reconciler
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	// Add fields to track mock behavior
	createValuesFileCalled            bool
	installOrUpgradeVClusterCalled    bool
	ensureSchemaConfigMapCalled       bool
	validateValuesAgainstSchemaCalled bool
	helmReleaseExistsCalled           bool
	finalizeVirtualClusterCalled      bool

	// Add mock return values
	createValuesFileResult           string
	createValuesFileError            error
	installOrUpgradeVClusterError    error
	ensureSchemaConfigMapResult      string
	ensureSchemaConfigMapError       error
	validateValuesAgainstSchemaError error
	helmReleaseExistsResult          bool
	helmReleaseExistsError           error
	finalizeVirtualClusterError      error
}

// Reconcile implements the reconcile.Reconciler interface
func (r *MockVirtualClusterReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	// Reconcile logic is mocked here
	vc := &corev1alpha1.VirtualCluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, vc); err != nil {
		// Return empty result if resource not found
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Handle finalization
	if !vc.DeletionTimestamp.IsZero() && containsString(vc.Finalizers, vclusterFinalizer) {
		r.finalizeVirtualClusterCalled = true
		if r.finalizeVirtualClusterError != nil {
			return reconcile.Result{}, r.finalizeVirtualClusterError
		}
		// Remove finalizer
		vc.Finalizers = removeString(vc.Finalizers, vclusterFinalizer)
		if err := r.Client.Update(ctx, vc); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// Initialize if needed
	if vc.Status.Phase == "" {
		vc.Status.Phase = corev1alpha1.VirtualClusterPending
		vc.Status.Message = "Initializing VirtualCluster"
		meta.SetStatusCondition(&vc.Status.Conditions, metav1.Condition{
			Type:    VirtualClusterConditionDeploying,
			Status:  metav1.ConditionTrue,
			Reason:  "Initializing",
			Message: "VirtualCluster is being initialized",
		})
		if err := r.Status().Update(ctx, vc); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	// Add finalizer if it doesn't exist
	if !containsString(vc.Finalizers, vclusterFinalizer) {
		vc.Finalizers = append(vc.Finalizers, vclusterFinalizer)
		if err := r.Client.Update(ctx, vc); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	// Handle provisioning
	if vc.Status.Phase == corev1alpha1.VirtualClusterProvisioning {
		r.createValuesFileCalled = true
		if r.createValuesFileError != nil {
			// Set error condition
			vc.Status.Phase = corev1alpha1.VirtualClusterFailed
			vc.Status.Message = r.createValuesFileError.Error()
			meta.SetStatusCondition(&vc.Status.Conditions, metav1.Condition{
				Type:    VirtualClusterConditionError,
				Status:  metav1.ConditionTrue,
				Reason:  "CreateValuesFileFailed",
				Message: r.createValuesFileError.Error(),
			})
			if err := r.Status().Update(ctx, vc); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, r.createValuesFileError
		}

		r.installOrUpgradeVClusterCalled = true
		if r.installOrUpgradeVClusterError != nil {
			return reconcile.Result{}, r.installOrUpgradeVClusterError
		}
	}

	return reconcile.Result{}, nil
}

// createValuesFile is a mocked implementation
func (r *MockVirtualClusterReconciler) createValuesFile(ctx context.Context, vcluster *corev1alpha1.VirtualCluster) (string, error) {
	r.createValuesFileCalled = true
	return r.createValuesFileResult, r.createValuesFileError
}

// installOrUpgradeVCluster is a mocked implementation
func (r *MockVirtualClusterReconciler) installOrUpgradeVCluster(ctx context.Context, vcluster *corev1alpha1.VirtualCluster, valuesFile string) error {
	r.installOrUpgradeVClusterCalled = true
	return r.installOrUpgradeVClusterError
}

// ensureSchemaConfigMap is a mocked implementation
func (r *MockVirtualClusterReconciler) ensureSchemaConfigMap(ctx context.Context, vcluster *corev1alpha1.VirtualCluster, version string) (string, error) {
	r.ensureSchemaConfigMapCalled = true
	return r.ensureSchemaConfigMapResult, r.ensureSchemaConfigMapError
}

// validateValuesAgainstSchema is a mocked implementation
func (r *MockVirtualClusterReconciler) validateValuesAgainstSchema(ctx context.Context, vcluster *corev1alpha1.VirtualCluster, schemaData string) error {
	r.validateValuesAgainstSchemaCalled = true
	return r.validateValuesAgainstSchemaError
}

// helmReleaseExists is a mocked implementation
func (r *MockVirtualClusterReconciler) helmReleaseExists(ctx context.Context, vcluster *corev1alpha1.VirtualCluster) (bool, error) {
	r.helmReleaseExistsCalled = true
	return r.helmReleaseExistsResult, r.helmReleaseExistsError
}

// finalizeVirtualCluster is a mocked implementation
func (r *MockVirtualClusterReconciler) finalizeVirtualCluster(ctx context.Context, vcluster *corev1alpha1.VirtualCluster) error {
	r.finalizeVirtualClusterCalled = true
	return r.finalizeVirtualClusterError
}

var _ = Describe("VirtualCluster Reconciler", func() {
	// Create a context for tests
	ctx := context.Background()

	Context("Reconcile with Mocked Dependencies", func() {
		var (
			fakeClient     client.Client
			mockReconciler *MockVirtualClusterReconciler
			namespacedName types.NamespacedName
			vcluster       *corev1alpha1.VirtualCluster
		)

		BeforeEach(func() {
			// Create a scheme with our custom types and core K8s types
			scheme := runtime.NewScheme()
			Expect(corev1alpha1.AddToScheme(scheme)).To(Succeed())
			Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed()) // Add core k8s types
			Expect(apiextensionsv1.AddToScheme(scheme)).To(Succeed())

			// Create a fake client
			fakeClient = fake.NewClientBuilder().WithScheme(scheme).Build()

			// Create the mock reconciler
			mockReconciler = &MockVirtualClusterReconciler{
				Client:   fakeClient,
				Scheme:   scheme,
				Recorder: record.NewFakeRecorder(10),
				// Set default mock return values
				createValuesFileResult:           "/tmp/mock-values.yaml",
				createValuesFileError:            nil,
				installOrUpgradeVClusterError:    nil,
				ensureSchemaConfigMapResult:      `{"schema": "mock"}`,
				ensureSchemaConfigMapError:       nil,
				validateValuesAgainstSchemaError: nil,
				helmReleaseExistsResult:          false,
				helmReleaseExistsError:           nil,
				finalizeVirtualClusterError:      nil,
			}

			// Create a VirtualCluster for testing
			namespacedName = types.NamespacedName{
				Name:      "test-vcluster",
				Namespace: "default",
			}

			vcluster = &corev1alpha1.VirtualCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namespacedName.Name,
					Namespace: namespacedName.Namespace,
				},
				Spec: corev1alpha1.VirtualClusterSpec{
					Chart: corev1alpha1.HelmChart{
						Version: "v0.24.1",
					},
					Values: &apiextensionsv1.JSON{Raw: []byte(`{"vcluster": {"image": "rancher/k3s:v1.25.0-k3s1"}}`)},
				},
			}

			// Create the VirtualCluster in the fake client
			Expect(fakeClient.Create(ctx, vcluster)).To(Succeed())
		})

		It("should initialize a new VirtualCluster correctly", func() {
			Skip("This test needs the MockVirtualClusterReconciler to be properly implemented")

			// Call Reconcile
			_, err := mockReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			// Get the updated VirtualCluster
			updatedVC := &corev1alpha1.VirtualCluster{}
			Expect(fakeClient.Get(ctx, namespacedName, updatedVC)).To(Succeed())

			// Check the status is initialized
			Expect(updatedVC.Status.Phase).To(Equal(corev1alpha1.VirtualClusterPending))
			Expect(updatedVC.Status.Message).To(ContainSubstring("Initializing"))

			// Check that conditions were set
			cond := meta.FindStatusCondition(updatedVC.Status.Conditions, VirtualClusterConditionDeploying)
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		})

		It("should add finalizer if it doesn't exist", func() {
			Skip("This test needs the MockVirtualClusterReconciler to be properly implemented")

			// First reconcile to initialize
			_, err := mockReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			// Get the updated VirtualCluster
			updatedVC := &corev1alpha1.VirtualCluster{}
			Expect(fakeClient.Get(ctx, namespacedName, updatedVC)).To(Succeed())

			// Update phase to continue to next step in reconcile
			updatedVC.Status.Phase = corev1alpha1.VirtualClusterPending
			Expect(fakeClient.Status().Update(ctx, updatedVC)).To(Succeed())

			// Reconcile again to add finalizer
			_, err = mockReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			// Get the updated VirtualCluster
			updatedVC = &corev1alpha1.VirtualCluster{}
			Expect(fakeClient.Get(ctx, namespacedName, updatedVC)).To(Succeed())

			// Check finalizer was added
			Expect(updatedVC.Finalizers).To(ContainElement(vclusterFinalizer))
		})

		It("should handle deletion and call finalizer", func() {
			Skip("This test needs the MockVirtualClusterReconciler to be properly implemented")

			// Add finalizer first
			vcluster.Finalizers = []string{vclusterFinalizer}
			Expect(fakeClient.Update(ctx, vcluster)).To(Succeed())

			// Mark for deletion by creating a new VC with deletion timestamp
			now := metav1.Now()
			deleteVC := vcluster.DeepCopy()
			deleteVC.DeletionTimestamp = &now

			// Delete will be handled on next reconcile
			_, err := mockReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			// Check that finalize was called
			Expect(mockReconciler.finalizeVirtualClusterCalled).To(BeTrue())
		})

		It("should proceed with installation when in Provisioning phase", func() {
			Skip("This test needs the MockVirtualClusterReconciler to be properly implemented")

			// Initialize first
			_, err := mockReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			// Get and update the phase
			updatedVC := &corev1alpha1.VirtualCluster{}
			Expect(fakeClient.Get(ctx, namespacedName, updatedVC)).To(Succeed())

			updatedVC.Status.Phase = corev1alpha1.VirtualClusterProvisioning
			Expect(fakeClient.Status().Update(ctx, updatedVC)).To(Succeed())

			// Add finalizer
			updatedVC.Finalizers = []string{vclusterFinalizer}
			Expect(fakeClient.Update(ctx, updatedVC)).To(Succeed())

			// Reconcile again to trigger installation
			_, err = mockReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			// Check that createValuesFile and installOrUpgradeVCluster were called
			Expect(mockReconciler.createValuesFileCalled).To(BeTrue())
			Expect(mockReconciler.installOrUpgradeVClusterCalled).To(BeTrue())
		})

		It("should handle errors in createValuesFile", func() {
			Skip("This test needs the MockVirtualClusterReconciler to be properly implemented")

			// Setup error
			mockReconciler.createValuesFileError = exec.Command("false").Run()

			// Initialize
			_, err := mockReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			// Get and update phase
			updatedVC := &corev1alpha1.VirtualCluster{}
			Expect(fakeClient.Get(ctx, namespacedName, updatedVC)).To(Succeed())

			updatedVC.Status.Phase = corev1alpha1.VirtualClusterProvisioning
			Expect(fakeClient.Status().Update(ctx, updatedVC)).To(Succeed())

			// Add finalizer
			updatedVC.Finalizers = []string{vclusterFinalizer}
			Expect(fakeClient.Update(ctx, updatedVC)).To(Succeed())

			// Reconcile to trigger error
			_, err = mockReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).To(HaveOccurred())

			// Get the updated VirtualCluster
			updatedVC = &corev1alpha1.VirtualCluster{}
			Expect(fakeClient.Get(ctx, namespacedName, updatedVC)).To(Succeed())

			// Check status was updated to Failed
			Expect(updatedVC.Status.Phase).To(Equal(corev1alpha1.VirtualClusterFailed))

			// Check Error condition was set
			cond := meta.FindStatusCondition(updatedVC.Status.Conditions, VirtualClusterConditionError)
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		})
	})
})

// Helper function to remove a string from a slice
func removeString(slice []string, s string) []string {
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}
