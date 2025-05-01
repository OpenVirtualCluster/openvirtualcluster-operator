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

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/OpenVirtualCluster/openvirtualcluster-operator/test/utils"
)

// exampleTimeout is the timeout for waiting for a VirtualCluster to become ready
const exampleTimeout = 5 * time.Minute

// examplePollingInterval is the interval at which we check if a VirtualCluster is ready
const examplePollingInterval = 5 * time.Second

var _ = Describe("Example VirtualClusters", Ordered, func() {
	// Define the example CRDs to test
	// We're not testing all examples to save time, but selecting key ones
	var exampleFiles = []string{
		"basic-virtualcluster.yaml",
		"custom-k8s-version.yaml",
		"resource-limits.yaml",
	}

	// Function to apply an example VirtualCluster
	var applyExample = func(filename string) error {
		examplePath := filepath.Join("examples", filename)
		cmd := exec.Command("kubectl", "apply", "-f", examplePath)
		_, err := utils.Run(cmd)
		return err
	}

	// Function to delete an example VirtualCluster
	var deleteExample = func(filename string) error {
		examplePath := filepath.Join("examples", filename)
		cmd := exec.Command("kubectl", "delete", "-f", examplePath, "--ignore-not-found=true")
		_, err := utils.Run(cmd)
		return err
	}

	// Function to clean up all example VirtualClusters
	var cleanupExamples = func() {
		for _, filename := range exampleFiles {
			_ = deleteExample(filename)
		}
	}

	// Function to check if examples directory exists
	var verifyExamplesExist = func() bool {
		_, err := os.Stat("examples")
		return err == nil
	}

	// Function to get VirtualCluster name from the file
	var getVCName = func(filename string) (string, string, error) {
		examplePath := filepath.Join("examples", filename)
		cmd := exec.Command("kubectl", "get", "-f", examplePath, "-o", "jsonpath={.metadata.name},{.metadata.namespace}")
		output, err := utils.Run(cmd)
		if err != nil {
			return "", "", err
		}

		// Split output to get name and namespace
		parts := strings.Split(output, ",")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("unexpected output format: %s", output)
		}

		return parts[0], parts[1], nil
	}

	// Function to check if a VirtualCluster is ready
	var isVirtualClusterReady = func(name, namespace string) (bool, error) {
		cmd := exec.Command("kubectl", "get", "virtualclusters", name, "-n", namespace, "-o", "jsonpath={.status.phase}")
		output, err := utils.Run(cmd)
		if err != nil {
			return false, err
		}

		return output == "Running", nil
	}

	// Run before all tests
	BeforeAll(func() {
		// Verify examples directory exists
		Expect(verifyExamplesExist()).To(BeTrue(), "Examples directory not found")

		// Clean up any existing examples
		cleanupExamples()
	})

	// Run after all tests
	AfterAll(func() {
		// Clean up examples
		cleanupExamples()
	})

	// Define the test cases
	for _, exampleFile := range exampleFiles {
		// Use a local variable to ensure the correct example is used in the closure
		example := exampleFile

		It(fmt.Sprintf("should successfully deploy and run %s", example), func() {
			// Apply the example VirtualCluster
			By(fmt.Sprintf("Applying example %s", example))
			err := applyExample(example)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to apply example %s", example))

			// Get VirtualCluster name and namespace
			vcName, vcNamespace, err := getVCName(example)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to get VirtualCluster name from %s", example))

			// Wait for the VirtualCluster to be ready
			By(fmt.Sprintf("Waiting for VirtualCluster %s in namespace %s to be ready", vcName, vcNamespace))
			Eventually(func() (bool, error) {
				return isVirtualClusterReady(vcName, vcNamespace)
			}, exampleTimeout, examplePollingInterval).Should(BeTrue(),
				fmt.Sprintf("VirtualCluster %s in namespace %s did not become ready in time", vcName, vcNamespace))

			// Verify the VirtualCluster pods are running
			By(fmt.Sprintf("Verifying VirtualCluster %s pods are running", vcName))
			cmd := exec.Command("kubectl", "get", "pods", "-n", vcNamespace, "-l", fmt.Sprintf("app=vcluster-%s", vcName), "-o", "jsonpath={.items[*].status.phase}")
			var podStatus string
			Eventually(func() (string, error) {
				podStatus, err = utils.Run(cmd)
				return podStatus, err
			}, exampleTimeout, examplePollingInterval).Should(ContainSubstring("Running"),
				fmt.Sprintf("VirtualCluster %s pods are not running", vcName))

			// Try to get kubeconfig from the vcluster
			By(fmt.Sprintf("Getting kubeconfig from VirtualCluster %s", vcName))
			kubeconfigCmd := exec.Command("kubectl", "get", "secret", fmt.Sprintf("%s-kubeconfig", vcName), "-n", vcNamespace, "-o", "jsonpath={.data.config}")
			var kubeconfigOutput string
			Eventually(func() (string, error) {
				kubeconfigOutput, err = utils.Run(kubeconfigCmd)
				return kubeconfigOutput, err
			}, exampleTimeout, examplePollingInterval).ShouldNot(BeEmpty(),
				fmt.Sprintf("Failed to get kubeconfig from VirtualCluster %s", vcName))

			// Clean up the VirtualCluster
			By(fmt.Sprintf("Cleaning up VirtualCluster %s", vcName))
			err = deleteExample(example)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to delete example %s", example))

			// Wait for the VirtualCluster to be deleted
			By(fmt.Sprintf("Waiting for VirtualCluster %s to be deleted", vcName))
			Eventually(func() bool {
				cmd := exec.Command("kubectl", "get", "virtualclusters", vcName, "-n", vcNamespace, "--ignore-not-found=true")
				output, _ := utils.Run(cmd)
				return output == ""
			}, exampleTimeout, examplePollingInterval).Should(BeTrue(),
				fmt.Sprintf("VirtualCluster %s was not deleted in time", vcName))
		})
	}
})

// Helper function to add example-specific tests if needed
func testExampleSpecificFeatures(exampleName, vcName, vcNamespace string) {
	switch exampleName {
	case "custom-k8s-version.yaml":
		// For custom K8s version, we could verify the correct version is running
		cmd := exec.Command("kubectl", "exec", "-n", vcNamespace,
			fmt.Sprintf("$(kubectl get pod -n %s -l app=vcluster-%s -o name | head -n 1)",
				vcNamespace, vcName),
			"--", "k3s", "--version")
		output, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to check K8s version")
		Expect(output).To(ContainSubstring("v1.28.2"), "Incorrect K8s version detected")
	}
}
