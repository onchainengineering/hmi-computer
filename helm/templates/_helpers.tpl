{{/*
Expand the name of the chart.
*/}}
{{- define "coder.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "coder.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "coder.selectorLabels" -}}
app.kubernetes.io/name: {{ include "coder.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "coder.labels" -}}
helm.sh/chart: {{ include "coder.chart" . }}
{{ include "coder.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Coder Docker image URI
*/}}
{{- define "coder.image" -}}
{{- if and (eq .Values.coder.image.tag "") (eq .Chart.AppVersion "0.1.0") -}}
{{ fail "You must specify the coder.image.tag value if you're installing the Helm chart directly from Git." }}
{{- end -}}
{{ .Values.coder.image.repo }}:{{ .Values.coder.image.tag | default (printf "v%v" .Chart.AppVersion) }}
{{- end }}

{{/*
Coder listen port (must be > 1024)
*/}}
{{- define "coder.port" }}
{{- if .Values.coder.tls.secretNames -}}
8443
{{- else -}}
8080
{{- end -}}
{{- end }}

{{/*
Coder service port
*/}}
{{- define "coder.servicePort" }}
{{- if .Values.coder.tls.secretNames -}}
443
{{- else -}}
80
{{- end -}}
{{- end }}

{{/*
Port name
*/}}
{{- define "coder.portName" }}
{{- if .Values.coder.tls.secretNames -}}
https
{{- else -}}
http
{{- end -}}
{{- end }}

{{/*
Scheme
*/}}
{{- define "coder.scheme" }}
{{- include "coder.portName" . | upper -}}
{{- end }}

{{/*
Coder volume definitions.
*/}}
{{- define "coder.volumes" }}
{{- if .Values.coder.tls.secretNames }}
volumes:
{{ range $secretName := .Values.coder.tls.secretNames -}}
- name: "tls-{{ $secretName }}"
  secret:
    secretName: {{ $secretName | quote }}
{{ end -}}
{{- else }}
volumes: []
{{- end }}
{{- end }}

{{/*
Coder volume mounts.
*/}}
{{- define "coder.volumeMounts" }}
{{- if .Values.coder.tls.secretNames }}
volumeMounts:
{{ range $secretName := .Values.coder.tls.secretNames -}}
- name: "tls-{{ $secretName }}"
  mountPath: "/etc/ssl/certs/coder/{{ $secretName }}"
  readOnly: true
{{ end }}
{{- else }}
volumeMounts: []
{{- end }}
{{- end }}

{{/*
Coder TLS environment variables.
*/}}
{{- define "coder.tlsEnv" }}
{{- if .Values.coder.tls.secretNames }}
- name: CODER_TLS_ENABLE
  value: "true"
- name: CODER_TLS_CERT_FILE
  value: "{{ range $idx, $secretName := .Values.coder.tls.secretNames -}}{{ if $idx }},{{ end }}/etc/ssl/certs/coder/{{ $secretName }}/tls.crt{{- end }}"
- name: CODER_TLS_KEY_FILE
  value: "{{ range $idx, $secretName := .Values.coder.tls.secretNames -}}{{ if $idx }},{{ end }}/etc/ssl/certs/coder/{{ $secretName }}/tls.key{{- end }}"
{{- end }}
{{- end }}

{{/*
Coder ingress wildcard hostname with the wildcard suffix stripped.
*/}}
{{- define "coder.ingressWildcardHost" -}}
{{/* This regex replace is required as the original input including the suffix
   * is not a legal ingress host. We need to remove the suffix and keep the
   * wildcard '*'.
   *
   *   - '\\*'     Starts with '*'
   *   - '[^.]*'   Suffix is 0 or more characters, '-suffix'
   *   - '('       Start domain capture group
   *   -   '\\.'     The domain should be separated with a '.' from the subdomain
   *   -   '.*'      Rest of the domain.
   *   - ')'       $1 is the ''.example.com'
   */}}
{{- regexReplaceAll "\\*[^.]*(\\..*)" .Values.coder.ingress.wildcardHost "*${1}" -}}
{{- end }}

{{/*
Fail on fully deprecated values or deprecated value combinations. This is
included at the top of coder.yaml.
*/}}
{{- define "coder.verifyDeprecated" }}
{{/*
Deprecated value coder.tls.secretName must not be used.
*/}}
{{- if .Values.coder.tls.secretName }}
{{ fail "coder.tls.secretName is deprecated, use coder.tls.secretNames instead." }}
{{- end }}
{{- end }}
