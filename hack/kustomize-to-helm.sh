#!/bin/bash

set -e

CHART_PATH="charts/openvirtualcluster-operator"

# Ensure temporary files are cleaned up
cleanup() {
  rm -f ${CHART_PATH}/templates/tmp-*.yaml
}
trap cleanup EXIT

# Function to create Helm template from Kustomize output
create_templates() {
  echo "Processing RBAC templates..."
  
  if [ -f "${CHART_PATH}/templates/rbac-kustomize.yaml" ]; then
    # Create RBAC template
    echo '{{- if .Values.rbac.create -}}' > ${CHART_PATH}/templates/rbac.yaml
    
    # Process RBAC template with basic replacements
    cat ${CHART_PATH}/templates/rbac-kustomize.yaml | \
      sed 's/namespace: system/namespace: {{ .Release.Namespace }}/g' | \
      sed 's/name: controller-manager/name: {{ include "openvirtualcluster-operator.fullname" . }}/g' | \
      sed 's/name: manager-role/name: {{ include "openvirtualcluster-operator.fullname" . }}-manager-role/g' | \
      sed 's/name: manager-rolebinding/name: {{ include "openvirtualcluster-operator.fullname" . }}-manager-rolebinding/g' | \
      sed 's/name: leader-election-role/name: {{ include "openvirtualcluster-operator.fullname" . }}-leader-election-role/g' | \
      sed 's/name: leader-election-rolebinding/name: {{ include "openvirtualcluster-operator.fullname" . }}-leader-election-rolebinding/g' | \
      sed 's/name: metrics-reader/name: {{ include "openvirtualcluster-operator.fullname" . }}-metrics-reader/g' | \
      sed 's/name: metrics-auth/name: {{ include "openvirtualcluster-operator.fullname" . }}-metrics-auth/g' | \
      sed 's/name: proxy-role/name: {{ include "openvirtualcluster-operator.fullname" . }}-proxy-role/g' | \
      sed 's/name: proxy-rolebinding/name: {{ include "openvirtualcluster-operator.fullname" . }}-proxy-rolebinding/g' \
      >> ${CHART_PATH}/templates/rbac.yaml
    
    echo '{{- end }}' >> ${CHART_PATH}/templates/rbac.yaml
    
    # Extract service account
    echo "Processing service account..."
    echo '{{- if .Values.serviceAccount.create -}}' > ${CHART_PATH}/templates/serviceaccount.yaml
    grep -A 10 'kind: ServiceAccount' ${CHART_PATH}/templates/rbac-kustomize.yaml | \
      sed 's/namespace: system/namespace: {{ .Release.Namespace }}/g' | \
      sed 's/name: controller-manager/name: {{ include "openvirtualcluster-operator.serviceAccountName" . }}/g' \
      >> ${CHART_PATH}/templates/serviceaccount.yaml
    echo '{{- end }}' >> ${CHART_PATH}/templates/serviceaccount.yaml
  fi
  
  echo "Processing manager deployment..."
  if [ -f "${CHART_PATH}/templates/manager-kustomize.yaml" ]; then
    # Create a new deployment.yaml file
    cat > ${CHART_PATH}/templates/deployment.yaml << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "openvirtualcluster-operator.fullname" . }}
  namespace: {{ .Release.Namespace }}
  labels:
    control-plane: controller-manager
    {{- include "openvirtualcluster-operator.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      control-plane: controller-manager
      {{- include "openvirtualcluster-operator.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        control-plane: controller-manager
        {{- include "openvirtualcluster-operator.selectorLabels" . | nindent 8 }}
    spec:
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      securityContext:
        {{- toYaml .Values.podSecurityContext | nindent 8 }}
      containers:
      - name: manager
        image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
        imagePullPolicy: {{ .Values.image.pullPolicy }}
        args:
          {{- if .Values.operator.leaderElection }}
          - --leader-elect
          {{- end }}
          - --health-probe-bind-address={{ .Values.operator.healthProbeBindAddress }}
          - --metrics-bind-address={{ .Values.operator.metricsBindAddress }}
          {{- if .Values.operator.metricsBindAddress }}
          - --metrics-secure={{ .Values.operator.secureMetrics }}
          {{- end }}
          {{- if .Values.operator.enableHTTP2 }}
          - --enable-http2
          {{- end }}
        securityContext:
          {{- toYaml .Values.securityContext | nindent 10 }}
        resources:
          {{- toYaml .Values.resources | nindent 10 }}
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8081
          initialDelaySeconds: 15
          periodSeconds: 20
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 10
      serviceAccountName: {{ include "openvirtualcluster-operator.serviceAccountName" . }}
      terminationGracePeriodSeconds: 10
EOF

    echo "Processing metrics service..."
    # Check if a metrics service exists
    if grep -q "kind: Service" ${CHART_PATH}/templates/manager-kustomize.yaml; then
      # Create metrics service
      cat > ${CHART_PATH}/templates/metrics-service.yaml << 'EOF'
{{- if .Values.metrics.enabled }}
apiVersion: v1
kind: Service
metadata:
  name: {{ include "openvirtualcluster-operator.fullname" . }}-metrics-service
  namespace: {{ .Release.Namespace }}
  labels:
    control-plane: controller-manager
    {{- include "openvirtualcluster-operator.labels" . | nindent 4 }}
spec:
  ports:
  - name: metrics
    port: 8443
    protocol: TCP
    targetPort: https
  selector:
    control-plane: controller-manager
    {{- include "openvirtualcluster-operator.selectorLabels" . | nindent 4 }}
{{- end }}
EOF
    fi
  fi
}

# Copy CRDs
echo "Generating Helm chart templates from Kustomize configurations..."
echo "Copying CRDs to chart directory..."
mkdir -p ${CHART_PATH}/crds
cp config/crd/bases/* ${CHART_PATH}/crds/ 2>/dev/null || true

# Generate RBAC and manager kustomize output
echo "Generating RBAC templates..."
mkdir -p ${CHART_PATH}/templates
${KUSTOMIZE:-./bin/kustomize} build config/rbac > ${CHART_PATH}/templates/rbac-kustomize.yaml

echo "Generating manager templates..."
${KUSTOMIZE:-./bin/kustomize} build config/manager > ${CHART_PATH}/templates/manager-kustomize.yaml

echo "Converting Kustomize output to Helm templates..."
create_templates

# Clean up kustomize output files
rm -f ${CHART_PATH}/templates/rbac-kustomize.yaml ${CHART_PATH}/templates/manager-kustomize.yaml

echo "Helm chart templates generated successfully!" 