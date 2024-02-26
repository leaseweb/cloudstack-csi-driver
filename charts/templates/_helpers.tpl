{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "csi.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "csi.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "csi.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "csi.labels" -}}
app.kubernetes.io/name: {{ include "csi.name" . }}
helm.sh/chart: {{ include "csi.chart" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}


{{/*
Create the name of the service account to use
*/}}
{{- define "csi.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default (include "csi.fullname" .) .Values.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
Create unified labels for csi components
*/}}
{{- define "csi.common.matchLabels" -}}
app: {{ template "csi.name" . }}
release: {{ .Release.Name }}
{{- end -}}

{{- define "csi.common.metaLabels" -}}
chart: {{ template "csi.chart" . }}
heritage: {{ .Release.Service }}
{{- if .Values.extraLabels }}
{{ toYaml .Values.extraLabels -}}
{{- end }}
{{- end -}}

{{- define "csi.controller.matchLabels" -}}
component: controller
{{ include "csi.common.matchLabels" . }}
{{- end -}}

{{- define "csi.controller.labels" -}}
{{ include "csi.controller.matchLabels" . }}
{{ include "csi.common.metaLabels" . }}
{{- end -}}

{{- define "csi.node.matchLabels" -}}
component: node
{{ include "csi.common.matchLabels" . }}
{{- end -}}

{{- define "csi.node.labels" -}}
{{ include "csi.node.matchLabels" . }}
{{ include "csi.common.metaLabels" . }}
{{- end -}}

{{/*
Create cloud-config makro.
*/}}
{{- define "cloudConfig" -}}
[Global]
{{- range $key, $value := .Values.cloudConfigData.global }}
{{ $key }} = {{ $value | quote }}
{{- end }}
{{- end -}}