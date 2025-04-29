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
	corev1alpha1 "github.com/prakashmishra1598/openvc/api/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
)

var _ = Describe("VirtualCluster Controller Utils", func() {
	var (
		ctx        context.Context
		reconciler *VirtualClusterReconciler
		testDir    string
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Create a temporary directory for test files
		var err error
		testDir, err = os.MkdirTemp("", "vcluster-test-")
		Expect(err).NotTo(HaveOccurred())

		reconciler = &VirtualClusterReconciler{
			Client:   k8sClient,
			Scheme:   k8sClient.Scheme(),
			Recorder: record.NewFakeRecorder(10),
		}
	})

	AfterEach(func() {
		// Clean up the temporary directory
		os.RemoveAll(testDir)
	})

	Context("Values Processing", func() {
		It("should correctly extract values from VirtualCluster", func() {
			// Create a test VirtualCluster with values
			vc := &corev1alpha1.VirtualCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-values-extraction",
					Namespace: "default",
				},
				Spec: corev1alpha1.VirtualClusterSpec{
					Chart: corev1alpha1.HelmChart{
						Version: "v0.24.1",
					},
					Values: &apiextensionsv1.JSON{Raw: []byte(`{
						"vcluster": {
							"image": "rancher/k3s:v1.25.0-k3s1",
							"extraArgs": ["--disable=traefik"]
						},
						"service": {
							"type": "ClusterIP"
						}
					}`)},
				},
			}

			// Get values
			values, err := vc.GetValues()
			Expect(err).NotTo(HaveOccurred())
			Expect(values).NotTo(BeNil())

			// Check that values were extracted correctly
			vclusterMap, ok := values["vcluster"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(vclusterMap["image"]).To(Equal("rancher/k3s:v1.25.0-k3s1"))

			serviceMap, ok := values["service"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(serviceMap["type"]).To(Equal("ClusterIP"))
		})
	})

	Context("File Operations", func() {
		It("should create and clean up temporary files", func() {
			// Create a test VirtualCluster
			vc := &corev1alpha1.VirtualCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-file-ops",
					Namespace: "default",
				},
				Spec: corev1alpha1.VirtualClusterSpec{
					Chart: corev1alpha1.HelmChart{
						Version: "v0.24.1",
					},
					Values: &apiextensionsv1.JSON{Raw: []byte(`{"vcluster": {"image": "rancher/k3s:v1.25.0-k3s1"}}`)},
				},
			}

			// Test the creation of values file
			valuesFile, err := reconciler.createValuesFile(ctx, vc)
			Expect(err).NotTo(HaveOccurred())
			Expect(valuesFile).NotTo(BeEmpty())

			// Check if file exists
			_, err = os.Stat(valuesFile)
			Expect(err).NotTo(HaveOccurred())

			// Clean up
			err = os.Remove(valuesFile)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Helm Commands", func() {
		// These tests are more difficult in a unit test environment and might require mocking

		It("should construct correct helm commands", func() {
			Skip("Helm command tests require more complex mocking or integration testing")

			// If you implement mocking for the helm commands, you can test here
			// Example: Check that helm add repo command is formatted correctly
			// Example: Check that helm install command includes all necessary flags
		})
	})

	Context("Schema Config Map", func() {
		It("should handle schema ConfigMap operations", func() {
			Skip("Schema ConfigMap tests require more complex mocking or integration testing")

			// If you implement mocking for the ConfigMap operations, you can test here
		})
	})

	Context("JSON Schema Validation", func() {
		It("should validate helm values against JSON schema", func() {
			Skip("JSON schema validation tests require actual schema data")

			// If you provide mock schema data, you can test validation logic here
		})
	})
})
