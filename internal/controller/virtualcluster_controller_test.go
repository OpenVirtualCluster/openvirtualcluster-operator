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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1alpha1 "github.com/prakashmishra1598/openvc/api/v1alpha1"
)

var _ = Describe("VirtualCluster Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		virtualcluster := &corev1alpha1.VirtualCluster{}

		BeforeEach(func() {
			// Make sure core types are registered in the test scheme
			err := clientgoscheme.AddToScheme(k8sClient.Scheme())
			Expect(err).NotTo(HaveOccurred())
			err = corev1.AddToScheme(k8sClient.Scheme())
			Expect(err).NotTo(HaveOccurred())

			By("creating the custom resource for the Kind VirtualCluster")
			err = k8sClient.Get(ctx, typeNamespacedName, virtualcluster)
			if err != nil && errors.IsNotFound(err) {
				resource := &corev1alpha1.VirtualCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: corev1alpha1.VirtualClusterSpec{
						Chart: corev1alpha1.HelmChart{
							Version: "v0.24.1",
						},
						Values: &apiextensionsv1.JSON{Raw: []byte(`{"vcluster": {"image": "rancher/k3s:v1.25.0-k3s1"}}`)},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &corev1alpha1.VirtualCluster{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance VirtualCluster")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			Skip("This test requires integration with a real k8s environment")

			By("Reconciling the created resource")
			controllerReconciler := &VirtualClusterReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(10),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Fetch the updated VirtualCluster
			updatedVC := &corev1alpha1.VirtualCluster{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedVC)
			Expect(err).NotTo(HaveOccurred())

			// Check that status is initialized
			Expect(updatedVC.Status.Phase).To(Equal(corev1alpha1.VirtualClusterPending))

			// Check that the finalizer is added
			Expect(updatedVC.Finalizers).To(ContainElement(vclusterFinalizer))
		})
	})

	Context("Status Conditions Handling", func() {
		It("should set and update status conditions correctly", func() {
			// Create a VirtualCluster instance
			vc := &corev1alpha1.VirtualCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-conditions",
					Namespace: "default",
				},
				Spec: corev1alpha1.VirtualClusterSpec{
					Chart: corev1alpha1.HelmChart{
						Version: "v0.24.1",
					},
					Values: &apiextensionsv1.JSON{Raw: []byte(`{"vcluster": {"image": "rancher/k3s:v1.25.0-k3s1"}}`)},
				},
				Status: corev1alpha1.VirtualClusterStatus{
					Phase: corev1alpha1.VirtualClusterPending,
				},
			}

			// Set a condition
			meta.SetStatusCondition(&vc.Status.Conditions, metav1.Condition{
				Type:    VirtualClusterConditionDeploying,
				Status:  metav1.ConditionTrue,
				Reason:  "Testing",
				Message: "Testing conditions",
			})

			// Verify condition is set
			cond := meta.FindStatusCondition(vc.Status.Conditions, VirtualClusterConditionDeploying)
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			Expect(cond.Reason).To(Equal("Testing"))

			// Update the condition
			meta.SetStatusCondition(&vc.Status.Conditions, metav1.Condition{
				Type:    VirtualClusterConditionDeploying,
				Status:  metav1.ConditionFalse,
				Reason:  "Completed",
				Message: "Deployment completed",
			})

			// Verify condition was updated
			cond = meta.FindStatusCondition(vc.Status.Conditions, VirtualClusterConditionDeploying)
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			Expect(cond.Reason).To(Equal("Completed"))
		})
	})

	Context("Values File Generation", func() {
		It("should create a valid values file", func() {
			vc := &corev1alpha1.VirtualCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-values",
					Namespace: "default",
				},
				Spec: corev1alpha1.VirtualClusterSpec{
					Chart: corev1alpha1.HelmChart{
						Version: "v0.24.1",
					},
					Values: &apiextensionsv1.JSON{Raw: []byte(`{"vcluster": {"image": "rancher/k3s:v1.25.0-k3s1"}}`)},
				},
			}

			reconciler := &VirtualClusterReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				Recorder: record.NewFakeRecorder(10),
			}

			// Test createValuesFile function
			valuesFile, err := reconciler.createValuesFile(ctx, vc)
			Expect(err).NotTo(HaveOccurred())
			Expect(valuesFile).NotTo(BeEmpty())

			// Verify file exists
			_, err = os.Stat(valuesFile)
			Expect(err).NotTo(HaveOccurred())

			// Clean up
			defer os.Remove(valuesFile)

			// Check content
			content, err := os.ReadFile(valuesFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("rancher/k3s:v1.25.0-k3s1"))
		})
	})

	Context("Schema Validation", func() {
		It("should validate values against schema", func() {
			Skip("Schema validation requires actual schema data from vCluster which might not be available in tests")
		})
	})

	Context("Finalizer Logic", func() {
		It("should handle finalization correctly", func() {
			Skip("Finalizer tests require more complex mocking")
		})
	})
})
