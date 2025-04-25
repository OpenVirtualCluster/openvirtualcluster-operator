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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"

	corev1alpha1 "github.com/prakashmishra1598/openvc/api/v1alpha1"
)

const (
	// Define a constant for the vCluster version
	vclusterVersion = "v0.24.1"
	// Define a constant for the vCluster chart name
	vclusterChart = "vcluster"
	// Define a constant for the vCluster chart repo
	vclusterRepo = "https://charts.loft.sh"
	// Finalizer for our resources
	vclusterFinalizer = "core.openvc.dev/finalizer"

	// ConditionTypes for VirtualCluster
	VirtualClusterConditionAvailable = "Available"
	VirtualClusterConditionDeploying = "Deploying"
	VirtualClusterConditionError     = "Error"
)

// VirtualClusterReconciler reconciles a VirtualCluster object
type VirtualClusterReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=core.openvc.dev,resources=virtualclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.openvc.dev,resources=virtualclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.openvc.dev,resources=virtualclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *VirtualClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling VirtualCluster", "namespace", req.Namespace, "name", req.Name)

	// Fetch the VirtualCluster instance
	vcluster := &corev1alpha1.VirtualCluster{}
	err := r.Get(ctx, req.NamespacedName, vcluster)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			logger.Info("VirtualCluster resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get VirtualCluster")
		return ctrl.Result{}, err
	}

	// Initialize status if needed
	if vcluster.Status.Phase == "" {
		vcluster.Status.Phase = corev1alpha1.VirtualClusterPending
		vcluster.Status.Message = "Initializing VirtualCluster"
		vcluster.Status.HelmChart = fmt.Sprintf("loft/%s", vclusterChart)
		vcluster.Status.HelmRelease = vcluster.Name

		// Set initial conditions
		meta.SetStatusCondition(&vcluster.Status.Conditions, metav1.Condition{
			Type:    VirtualClusterConditionDeploying,
			Status:  metav1.ConditionTrue,
			Reason:  "Initializing",
			Message: "VirtualCluster is being initialized",
		})

		if err := r.Status().Update(ctx, vcluster); err != nil {
			logger.Error(err, "Failed to update VirtualCluster status")
			return ctrl.Result{}, err
		}

		// Return here to process the rest in the next reconciliation
		return ctrl.Result{Requeue: true}, nil
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(vcluster, vclusterFinalizer) {
		controllerutil.AddFinalizer(vcluster, vclusterFinalizer)
		err = r.Update(ctx, vcluster)
		if err != nil {
			logger.Error(err, "Failed to add finalizer to VirtualCluster")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Check if the VirtualCluster instance is marked to be deleted
	if !vcluster.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is being deleted
		logger.Info("Deleting VirtualCluster", "namespace", req.Namespace, "name", req.Name)

		// Update status to Deleting
		if vcluster.Status.Phase != corev1alpha1.VirtualClusterDeleting {
			vcluster.Status.Phase = corev1alpha1.VirtualClusterDeleting
			vcluster.Status.Message = "Deleting VirtualCluster"
			if err := r.Status().Update(ctx, vcluster); err != nil {
				logger.Error(err, "Failed to update VirtualCluster status")
				return ctrl.Result{}, err
			}
		}

		if controllerutil.ContainsFinalizer(vcluster, vclusterFinalizer) {
			// Run finalization logic
			if err := r.finalizeVirtualCluster(ctx, vcluster); err != nil {
				// If finalization fails, return error so that we can retry
				logger.Error(err, "Failed to finalize VirtualCluster")

				// Update Error condition
				meta.SetStatusCondition(&vcluster.Status.Conditions, metav1.Condition{
					Type:    VirtualClusterConditionError,
					Status:  metav1.ConditionTrue,
					Reason:  "FinalizationFailed",
					Message: fmt.Sprintf("Error during finalization: %v", err),
				})

				if err := r.Status().Update(ctx, vcluster); err != nil {
					logger.Error(err, "Failed to update VirtualCluster status")
					return ctrl.Result{}, err
				}

				return ctrl.Result{}, err
			}

			// Remove finalizer once finalization is done
			controllerutil.RemoveFinalizer(vcluster, vclusterFinalizer)
			err = r.Update(ctx, vcluster)
			if err != nil {
				logger.Error(err, "Failed to remove finalizer from VirtualCluster")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Update status to Provisioning
	if vcluster.Status.Phase == corev1alpha1.VirtualClusterPending {
		vcluster.Status.Phase = corev1alpha1.VirtualClusterProvisioning
		vcluster.Status.Message = "Deploying VirtualCluster using Helm"

		meta.SetStatusCondition(&vcluster.Status.Conditions, metav1.Condition{
			Type:    VirtualClusterConditionDeploying,
			Status:  metav1.ConditionTrue,
			Reason:  "Deploying",
			Message: "VirtualCluster is being deployed",
		})

		if err := r.Status().Update(ctx, vcluster); err != nil {
			logger.Error(err, "Failed to update VirtualCluster status")
			return ctrl.Result{}, err
		}
	}

	// Create the values file
	valuesFile, err := r.createValuesFile(ctx, vcluster)
	if err != nil {
		logger.Error(err, "Failed to create values file")

		// Update status to Failed
		vcluster.Status.Phase = corev1alpha1.VirtualClusterFailed
		vcluster.Status.Message = fmt.Sprintf("Failed to create values file: %v", err)

		meta.SetStatusCondition(&vcluster.Status.Conditions, metav1.Condition{
			Type:    VirtualClusterConditionError,
			Status:  metav1.ConditionTrue,
			Reason:  "ValuesFileCreationFailed",
			Message: fmt.Sprintf("Failed to create values file: %v", err),
		})

		if err := r.Status().Update(ctx, vcluster); err != nil {
			logger.Error(err, "Failed to update VirtualCluster status")
		}

		return ctrl.Result{}, err
	}

	// Install or upgrade the vCluster
	err = r.installOrUpgradeVCluster(ctx, vcluster, valuesFile)
	if err != nil {
		logger.Error(err, "Failed to install or upgrade vCluster")

		// Update status to Failed
		vcluster.Status.Phase = corev1alpha1.VirtualClusterFailed
		vcluster.Status.Message = fmt.Sprintf("Failed to install or upgrade vCluster: %v", err)

		meta.SetStatusCondition(&vcluster.Status.Conditions, metav1.Condition{
			Type:    VirtualClusterConditionError,
			Status:  metav1.ConditionTrue,
			Reason:  "HelmOperationFailed",
			Message: fmt.Sprintf("Failed to install or upgrade vCluster: %v", err),
		})

		meta.SetStatusCondition(&vcluster.Status.Conditions, metav1.Condition{
			Type:    VirtualClusterConditionAvailable,
			Status:  metav1.ConditionFalse,
			Reason:  "HelmOperationFailed",
			Message: "VirtualCluster is not available due to Helm operation failure",
		})

		if err := r.Status().Update(ctx, vcluster); err != nil {
			logger.Error(err, "Failed to update VirtualCluster status")
		}

		// Record an event
		r.Recorder.Event(vcluster, corev1.EventTypeWarning, "InstallFailed",
			fmt.Sprintf("Failed to install or upgrade vCluster: %v", err))

		return ctrl.Result{}, err
	}

	// Check if the status should be updated to Running
	if vcluster.Status.Phase != corev1alpha1.VirtualClusterRunning {
		vcluster.Status.Phase = corev1alpha1.VirtualClusterRunning
		vcluster.Status.Message = "VirtualCluster is running"

		meta.SetStatusCondition(&vcluster.Status.Conditions, metav1.Condition{
			Type:    VirtualClusterConditionDeploying,
			Status:  metav1.ConditionFalse,
			Reason:  "Deployed",
			Message: "VirtualCluster has been deployed",
		})

		meta.SetStatusCondition(&vcluster.Status.Conditions, metav1.Condition{
			Type:    VirtualClusterConditionAvailable,
			Status:  metav1.ConditionTrue,
			Reason:  "Running",
			Message: "VirtualCluster is available and running",
		})

		meta.SetStatusCondition(&vcluster.Status.Conditions, metav1.Condition{
			Type:    VirtualClusterConditionError,
			Status:  metav1.ConditionFalse,
			Reason:  "NoError",
			Message: "No errors detected",
		})

		if err := r.Status().Update(ctx, vcluster); err != nil {
			logger.Error(err, "Failed to update VirtualCluster status")
			return ctrl.Result{}, err
		}

		// Record an event
		r.Recorder.Event(vcluster, corev1.EventTypeNormal, "Deployed",
			"VirtualCluster has been successfully deployed")
	}

	return ctrl.Result{}, nil
}

// createValuesFile creates a temporary values file for the vCluster Helm chart
func (r *VirtualClusterReconciler) createValuesFile(ctx context.Context, vcluster *corev1alpha1.VirtualCluster) (string, error) {
	logger := log.FromContext(ctx)

	// Convert the values to YAML
	valuesYAML, err := yaml.Marshal(vcluster.Spec.Values.Config)
	if err != nil {
		logger.Error(err, "Failed to marshal values to YAML")
		return "", err
	}

	// Create the values file
	valuesFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("values-%s-%s.yaml", vcluster.Name, vcluster.Namespace))
	logger.Info("Writing values file", "path", valuesFilePath)
	err = os.WriteFile(valuesFilePath, valuesYAML, 0644)
	if err != nil {
		logger.Error(err, "Failed to write values file")
		return "", err
	}

	logger.Info("Created values file", "path", valuesFilePath)
	return valuesFilePath, nil
}

// installOrUpgradeVCluster installs or upgrades the vCluster using Helm
func (r *VirtualClusterReconciler) installOrUpgradeVCluster(ctx context.Context, vcluster *corev1alpha1.VirtualCluster, valuesFile string) error {
	logger := log.FromContext(ctx)
	logger.Info("Installing or upgrading vCluster", "namespace", vcluster.Namespace, "name", vcluster.Name)

	// Check if the Helm release exists
	exists, err := r.helmReleaseExists(ctx, vcluster)
	if err != nil {
		return err
	}

	// Prepare the Helm command
	var cmd *exec.Cmd
	releaseName := vcluster.Name
	namespace := vcluster.Namespace

	// Add the vCluster repo if not exists
	addRepoCmd := exec.Command("helm", "repo", "add", "loft", vclusterRepo)
	if output, err := addRepoCmd.CombinedOutput(); err != nil {
		logger.Error(err, "Failed to add Helm repo", "output", string(output))
		return err
	}

	// Update the Helm repos
	updateRepoCmd := exec.Command("helm", "repo", "update")
	if output, err := updateRepoCmd.CombinedOutput(); err != nil {
		logger.Error(err, "Failed to update Helm repos", "output", string(output))
		return err
	}

	if exists {
		logger.Info("Upgrading the release", "release", releaseName)
		// Upgrade the release
		cmd = exec.Command(
			"helm", "upgrade",
			releaseName,
			fmt.Sprintf("loft/%s", vclusterChart),
			"--version", vclusterVersion,
			"--namespace", namespace,
			"--values", valuesFile,
		)
	} else {
		logger.Info("Installing the release", "release", releaseName)
		// Install the release
		cmd = exec.Command(
			"helm", "install",
			releaseName,
			fmt.Sprintf("loft/%s", vclusterChart),
			"--version", vclusterVersion,
			"--namespace", namespace,
			"--create-namespace",
			"--values", valuesFile,
		)
	}

	// Execute the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error(err, "Failed to execute Helm command", "output", string(output))
		return fmt.Errorf("failed to execute Helm command: %v, output: %s", err, string(output))
	}

	logger.Info("Successfully executed Helm command", "output", string(output))
	return nil
}

// helmReleaseExists checks if a Helm release exists
func (r *VirtualClusterReconciler) helmReleaseExists(ctx context.Context, vcluster *corev1alpha1.VirtualCluster) (bool, error) {
	logger := log.FromContext(ctx)

	// Use helm list command to check if release exists
	cmd := exec.Command(
		"helm", "list",
		"--namespace", vcluster.Namespace,
		"--filter", vcluster.Name,
		"--output", "json",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error(err, "Failed to list Helm releases", "output", string(output))
		return false, err
	}

	logger.Info("Helm list output", "output", string(output))

	// Parse the output
	var releases []map[string]interface{}
	if err := json.Unmarshal(output, &releases); err != nil {
		logger.Error(err, "Failed to parse Helm list output")
		return false, err
	}

	// Check if our release is in the list
	for _, release := range releases {
		name, ok := release["name"].(string)
		if ok && name == vcluster.Name {
			return true, nil
		}
	}

	return false, nil
}

// finalizeVirtualCluster handles deletion of the VirtualCluster resource
func (r *VirtualClusterReconciler) finalizeVirtualCluster(ctx context.Context, vcluster *corev1alpha1.VirtualCluster) error {
	logger := log.FromContext(ctx)
	logger.Info("Finalizing VirtualCluster", "namespace", vcluster.Namespace, "name", vcluster.Name)

	// Use helm uninstall to delete the release
	cmd := exec.Command(
		"helm", "uninstall",
		vcluster.Name,
		"--namespace", vcluster.Namespace,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// If the error indicates that the release is not found, we can consider it already deleted
		if strings.Contains(string(output), "not found") {
			logger.Info("Helm release already deleted", "output", string(output))
			return nil
		}
		logger.Error(err, "Failed to uninstall Helm release", "output", string(output))
		return err
	}

	// Delete the values file if it exists
	valuesFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("values-%s-%s.yaml", vcluster.Name, vcluster.Namespace))
	if _, err := os.Stat(valuesFilePath); err == nil {
		if err := os.Remove(valuesFilePath); err != nil {
			logger.Error(err, "Failed to delete values file", "path", valuesFilePath)
			// Don't return error as this is not critical
		}
	}

	logger.Info("Successfully uninstalled Helm release", "output", string(output))
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.VirtualCluster{}).
		Named("virtualcluster").
		Complete(r)
}
