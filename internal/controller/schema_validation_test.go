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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	corev1alpha1 "github.com/OpenVirtualCluster/openvirtualcluster-operator/api/v1alpha1"
)

var _ = Describe("Schema Validation", func() {
	var (
		ctx        context.Context
		reconciler *VirtualClusterReconciler
		scheme     *runtime.Scheme
	)

	// Sample JSON schema
	const sampleSchema = `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"vcluster": {
				"type": "object",
				"properties": {
					"image": {
						"type": "string"
					},
					"extraArgs": {
						"type": "array",
						"items": {
							"type": "string"
						}
					}
				},
				"required": ["image"]
			},
			"service": {
				"type": "object",
				"properties": {
					"type": {
						"type": "string",
						"enum": ["ClusterIP", "NodePort", "LoadBalancer"]
					}
				}
			}
		}
	}`

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(corev1alpha1.AddToScheme(scheme)).To(Succeed())

		// Setup fake client and reconciler
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		reconciler = &VirtualClusterReconciler{
			Client:   fakeClient,
			Scheme:   scheme,
			Recorder: record.NewFakeRecorder(10),
		}
	})

	Context("validateValuesAgainstSchema", func() {
		It("should validate correct values successfully", func() {
			// Create a VirtualCluster with valid values
			vc := &corev1alpha1.VirtualCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-schema-test",
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

			// Validate against schema
			err := reconciler.validateValuesAgainstSchema(ctx, vc, sampleSchema)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject values that violate the schema", func() {
			// Create a VirtualCluster with invalid values (missing required image)
			vc := &corev1alpha1.VirtualCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-schema-test",
					Namespace: "default",
				},
				Spec: corev1alpha1.VirtualClusterSpec{
					Chart: corev1alpha1.HelmChart{
						Version: "v0.24.1",
					},
					Values: &apiextensionsv1.JSON{Raw: []byte(`{
						"vcluster": {
							"extraArgs": ["--disable=traefik"]
						},
						"service": {
							"type": "ClusterIP"
						}
					}`)},
				},
			}

			// Validate against schema - should fail due to missing required field
			err := reconciler.validateValuesAgainstSchema(ctx, vc, sampleSchema)
			Expect(err).To(HaveOccurred())
		})

		It("should reject values with incorrect type", func() {
			// Create a VirtualCluster with invalid value types
			vc := &corev1alpha1.VirtualCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-type-test",
					Namespace: "default",
				},
				Spec: corev1alpha1.VirtualClusterSpec{
					Chart: corev1alpha1.HelmChart{
						Version: "v0.24.1",
					},
					Values: &apiextensionsv1.JSON{Raw: []byte(`{
						"vcluster": {
							"image": "rancher/k3s:v1.25.0-k3s1",
							"extraArgs": 123  
						},
						"service": {
							"type": "ClusterIP"
						}
					}`)},
				},
			}

			// Validate against schema - should fail due to wrong type
			err := reconciler.validateValuesAgainstSchema(ctx, vc, sampleSchema)
			Expect(err).To(HaveOccurred())
		})

		It("should reject values with invalid enum values", func() {
			// Create a VirtualCluster with invalid enum value
			vc := &corev1alpha1.VirtualCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-enum-test",
					Namespace: "default",
				},
				Spec: corev1alpha1.VirtualClusterSpec{
					Chart: corev1alpha1.HelmChart{
						Version: "v0.24.1",
					},
					Values: &apiextensionsv1.JSON{Raw: []byte(`{
						"vcluster": {
							"image": "rancher/k3s:v1.25.0-k3s1"
						},
						"service": {
							"type": "InvalidType"
						}
					}`)},
				},
			}

			// Validate against schema - should fail due to invalid enum value
			err := reconciler.validateValuesAgainstSchema(ctx, vc, sampleSchema)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("ensureSchemaConfigMap", func() {
		It("should handle schema ConfigMap operations", func() {
			Skip("Schema ConfigMap tests require more complex mocking or integration testing")

			// If you implement mocking for the ConfigMap operations, you can test here
		})
	})
})
