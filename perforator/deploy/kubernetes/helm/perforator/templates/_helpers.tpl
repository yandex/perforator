{{/*
Expand the name of the chart.
*/}}
{{- define "perforator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "perforator.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "perforator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "perforator.labels" -}}
helm.sh/chart: {{ include "perforator.chart" . }}
{{ include "perforator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "perforator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "perforator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Agent selector labels
*/}}
{{- define "perforator.agent.selectorLabels" -}}
perforator.component: agent
{{- end }}

{{/*
Storage selector labels
*/}}
{{- define "perforator.storage.selectorLabels" -}}
perforator.component: storage
{{- end }}

{{/*
Proxy selector labels
*/}}
{{- define "perforator.proxy.selectorLabels" -}}
perforator.component: proxy
{{- end }}

{{/*
Web selector labels
*/}}
{{- define "perforator.web.selectorLabels" -}}
perforator.component: web
{{- end }}

{{/*
gc selector labels
*/}}
{{- define "perforator.gc.selectorLabels" -}}
perforator.component: gc
{{- end }}

{{/*
Offline processing selector labels
*/}}
{{- define "perforator.offlineprocessing.selectorLabels" -}}
perforator.component: offlineprocessing
{{- end }}

{{/*
PostgreSQL migration job selector labels
*/}}
{{- define "perforator.migrationspg.selectorLabels" -}}
app.kubernetes.io/component: migrations-pg
{{- end }}

{{/*
ClickHouse migration job selector labels
*/}}
{{- define "perforator.migrationsch.selectorLabels" -}}
app.kubernetes.io/component: migrations-ch
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "perforator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "perforator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
//////////////////////////////////////////////////////////////////////////////////////////// 
*/}}

{{/*
Return effective PostgreSQL endpoints.
*/}}
{{- define "perforator.postgresql.endpoints" -}}
{{- if .Values.testing.enableTestingDatabases -}}
- host: {{ printf "%s-postgresql" .Release.Name | quote }}
  port: {{ print 5432 }}
{{- else -}}
{{ toYaml .Values.databases.postgresql.endpoints }}
{{- end -}}
{{- end }}

{{/*
Return effective ClickHouse endpoints.
*/}}
{{- define "perforator.clickhouse.endpoints" -}}
{{- if .Values.testing.enableTestingDatabases -}}
{{- $host := printf "%s-clickhouse" .Release.Name -}}
{{- $port := "9440" -}}
- {{ printf "%s:%s" $host $port | quote }}
{{- else -}}
{{ toYaml .Values.databases.clickhouse.replicas }}
{{- end -}}
{{- end }}

{{/*
Return effective S3 endpoint.
*/}}
{{- define "perforator.s3.endpoint" -}}
{{- if .Values.testing.enableTestingDatabases -}}
{{- $host := printf "%s-minio" .Release.Name -}}
{{- $port := "9000" -}}
{{ printf "%s:%s" $host $port | quote }}
{{- else -}}
{{ .Values.databases.s3.endpoint | quote }}
{{- end -}}
{{- end }}

{{/*
//////////////////////////////////////////////////////////////////////////////////////////// 
*/}}

{{/*
Create secretKeyRef for one of databases
*/}}

{{- define "perforator.secretKeyRef.postgresql" -}}
{{- if .Values.databases.postgresql.password -}}
name: {{ include "perforator.fullname" . }}-postgresql-password
key: password
{{- else if (and .Values.databases.postgresql.secretName .Values.databases.postgresql.secretKey) -}}
name: {{ .Values.databases.postgresql.secretName }}
key: {{ .Values.databases.postgresql.secretKey }}
{{- else -}}
{{- printf "Specify postgresql secret name and password key name, your values: secret: %s field: %s" .Values.databases.postgresql.secretName .Values.databases.postgresql.secretKey | fail }}
{{- end }}
{{- end }}

{{- define "perforator.secretKeyRef.clickhouse" -}}
{{- if .Values.databases.clickhouse.password -}}
name: {{ include "perforator.fullname" . }}-clickhouse-password
key: password
{{- else if (and .Values.databases.clickhouse.secretName .Values.databases.clickhouse.secretKey) -}}
name: {{ .Values.databases.clickhouse.secretName }}
key: {{ .Values.databases.clickhouse.secretKey }}
{{- else -}}
{{- printf "Specify clickhouse secret name and password key name, your values: secret: %s field: %s" .Values.databases.clickhouse.secretName .Values.databases.clickhouse.secretKey | fail }}
{{- end }}
{{- end }}

{{- define "perforator.secretName.s3" -}}
{{- if and .Values.databases.s3.accessKey .Values.databases.s3.secretKey -}}
{{ include "perforator.fullname" . }}-storage-s3-keys
{{- else if .Values.databases.s3.secretName -}}
{{ .Values.databases.s3.secretName }}
{{- else -}}
{{- "Specify s3 access key and secret key or s3 secret name, containing this values" | fail }}
{{- end }}
{{- end }}

{{/*
//////////////////////////////////////////////////////////////////////////////////////////// 
*/}}

{{/*
ClickHouse TLS Secret Name
*/}}
{{- define "perforator.clickhouse.tlsSecretName" -}}
{{- if .Values.databases.clickhouse.tls.enabled -}}
    {{- .Values.databases.clickhouse.tls.existingSecret -}}
{{- else -}}
    {{- printf "" -}}
{{- end -}}
{{- end -}}

{{/*
Return the path to the CA certificate file trusted by ClickHouse.
*/}}
{{- define "perforator.clickhouse.tlsCACert" -}}
{{- if .Values.databases.clickhouse.tls.enabled -}}
    {{- if and .Values.databases.clickhouse.tls.existingSecret .Values.databases.clickhouse.tls.certCAFilename -}}
        {{- printf "/etc/perforator/certificates/clickhouse/%s" .Values.databases.clickhouse.tls.certCAFilename -}}
    {{- else -}}
        {{- printf "" -}}
    {{- end -}}
{{- else -}}
    {{- printf "" -}}
{{- end -}}
{{- end -}}

{{/*
S3 TLS Secret Name
*/}}
{{- define "perforator.s3.tlsSecretName" -}}
{{- if .Values.databases.s3.tls.enabled -}}
    {{- .Values.databases.s3.tls.existingSecret -}}
{{- else -}}
    {{- printf "" -}}
{{- end -}}
{{- end -}}

{{/*
Return the path to the CA certificate file trusted by S3.
*/}}
{{- define "perforator.s3.tlsCACert" -}}
{{- if .Values.databases.s3.tls.enabled -}}
    {{- if and .Values.databases.s3.tls.existingSecret .Values.databases.s3.tls.certCAFilename -}}
        {{- printf "/etc/perforator/certificates/s3/%s" .Values.databases.s3.tls.certCAFilename -}}
    {{- else -}}
        {{- printf "" -}}
    {{- end -}}
{{- else -}}
    {{- printf "" -}}
{{- end -}}
{{- end -}}

{{/*
//////////////////////////////////////////////////////////////////////////////////////////// 
*/}}

{{- define "perforator.storage.tlsSecretName" -}}
{{- if .Values.storage.tls.enabled -}}
    {{- if or .Values.storage.tls.autoGenerated .Values.storageAgentTLS.autoGenerated -}}
        {{- printf "%s-storage-crt" (include "perforator.fullname" .) -}}
    {{- else -}}
        {{- $secret := coalesce .Values.storage.tls.existingSecret .Values.storageAgentTLS.storage.existingSecret -}}
        {{- required "Existing secret with certificates for storage must be specified when autoGenerated option is turned off" $secret | printf "%s" -}}
    {{- end -}}
{{- else -}}
    {{- printf "" -}}
{{- end -}}
{{- end -}}

{{/*
Return the path to the perforator storage certificate file.
Explicitly returns "" if TLS is turned off.
*/}}
{{- define "perforator.storage.tlsCert" -}}
{{- if .Values.storage.tls.enabled -}}
    {{- if or .Values.storage.tls.autoGenerated .Values.storageAgentTLS.autoGenerated -}}
        {{- printf "/etc/perforator/certificates/%s" "tls.crt" -}}
    {{- else -}}
        {{- $filename := coalesce .Values.storage.tls.certFilename .Values.storageAgentTLS.storage.certFilename -}}
        {{- required "Certificate filename for storage must be specified when autoGenerated option is turned off" $filename | printf "/etc/perforator/certificates/%s" -}}
    {{- end -}}
{{- else -}}
    {{- printf "" -}}
{{- end -}}
{{- end -}}

{{/*
Return the path to the perforator storage certificate key file.
Explicitly returns "" if TLS is turned off.
*/}}
{{- define "perforator.storage.tlsCertKey" -}}
{{- if .Values.storage.tls.enabled -}}
    {{- if or .Values.storage.tls.autoGenerated .Values.storageAgentTLS.autoGenerated -}}
        {{- printf "/etc/perforator/certificates/%s" "tls.key" -}}
    {{- else -}}
        {{- $certKeyFilename := coalesce .Values.storage.tls.certKeyFilename .Values.storageAgentTLS.storage.certKeyFilename -}}
        {{- required "Certificate Key filename for storage must be specified when autoGenerated option is turned off" $certKeyFilename | printf "/etc/perforator/certificates/%s" -}}
    {{- end -}}
{{- else -}}
    {{- printf "" -}}
{{- end -}}
{{- end -}}

{{/*
Return the path to the CA certificate file trusted by the perforator storage.
Explicitly returns "" if TLS is turned off.
*/}}
{{- define "perforator.storage.tlsCACert" -}}
{{- if .Values.storage.tls.enabled -}}
    {{- if or .Values.storage.tls.autoGenerated .Values.storageAgentTLS.autoGenerated -}}
        {{- printf "/etc/perforator/certificates/%s" "ca.crt" -}}
    {{- else if .Values.storage.tls.certCAFilename -}}
        {{- printf "/etc/perforator/certificates/%s" .Values.storage.tls.certCAFilename -}}
    {{- else -}}
        {{- printf "" -}}
    {{- end -}}
{{- else -}}
    {{- printf "" -}}
{{- end -}}
{{- end -}}

{{/*
//////////////////////////////////////////////////////////////////////////////////////////// 
*/}}

{{/*
Return the path to the perforator agent certificate file.
Explicitly returns "" if TLS is turned off.
*/}}
{{- define "perforator.agent.tlsSecretName" -}}
{{- if .Values.agent.tls.enabled -}}
    {{- if .Values.agent.tls.autoGenerated -}}
        {{- printf "%s-agent-crt" (include "perforator.fullname" .) -}}
    {{- else -}}
        {{- if and .Values.storageAgentTLS.storage.certCAFilename (eq .Values.agent.tls.existingSecret "") -}}
        {{ .Values.storageAgentTLS.storage.existingSecret }}
        {{- else -}}
        {{- if and (eq .Values.agent.tls.existingSecret "") (not (and (eq .Values.agent.tls.certFilename "") (eq .Values.agent.tls.certKeyFilename "") (eq .Values.agent.tls.certCAFilename ""))) -}}
            {{- fail "Error: When no existing tls secret is provided for perforator agent, certFilename, certKeyFilename and certCAFilename must be empty" -}}
        {{- end -}}
        {{- /* can be empty, if so we do not mount secret */ -}}
        {{ .Values.agent.tls.existingSecret }}
        {{- end -}}
    {{- end -}}
{{- else -}}
    {{- printf "" -}}
{{- end -}}
{{- end -}}

{{/*
Return the path to the perforator agent certificate file.
Explicitly returns "" if TLS is turned off.
*/}}
{{- define "perforator.agent.tlsCert" -}}
{{- if .Values.agent.tls.enabled -}}
    {{- if .Values.agent.tls.autoGenerated -}}
        {{- if .Values.storage.tls.verifyClient -}}
            {{- printf "/etc/perforator/certificates/%s" "tls.crt" -}}
        {{- else -}}
            {{- printf "" -}}
        {{- end -}}
    {{- else -}}
        {{- if .Values.agent.tls.certFilename -}}
            {{- printf "/etc/perforator/certificates/%s" .Values.agent.tls.certFilename -}}
        {{- else -}}
            {{- printf "" -}}
        {{- end -}}
    {{- end -}}
{{- else -}}
    {{- printf "" -}}
{{- end -}}
{{- end -}}

{{/*
Return the path to the perforator agent certificate key file.
Explicitly returns "" if TLS is turned off.
*/}}
{{- define "perforator.agent.tlsCertKey" -}}
{{- if .Values.agent.tls.enabled -}}
    {{- if .Values.agent.tls.autoGenerated -}}
        {{- if .Values.storage.tls.verifyClient -}}
            {{- printf "/etc/perforator/certificates/%s" "tls.key" -}}
        {{- else -}}
            {{- printf "" -}}
        {{- end -}}
    {{- else -}}
        {{- if .Values.agent.tls.certFilename -}}
            {{- printf "/etc/perforator/certificates/%s" .Values.agent.tls.certKeyFilename -}}
        {{- else -}}
            {{- printf "" -}}
        {{- end -}}
    {{- end -}}
{{- else -}}
    {{- printf "" -}}
{{- end -}}
{{- end -}}

{{/*
Return the path to the CA certificate file trusted by the perforator agent.
Explicitly returns "" if TLS is turned off.
*/}}
{{- define "perforator.agent.tlsCACert" -}}
{{- if .Values.agent.tls.enabled -}}
    {{- $certCAFilename := coalesce .Values.agent.tls.certCAFilename .Values.storageAgentTLS.storage.certCAFilename -}}
    {{- if .Values.agent.tls.autoGenerated -}}
        {{- printf "/etc/perforator/certificates/%s" "ca.crt" -}}
    {{- else if $certCAFilename -}}
        {{- printf "/etc/perforator/certificates/%s" $certCAFilename -}}
    {{- else -}}
        {{- printf "" -}}
    {{- end -}}
{{- else -}}
    {{- printf "" -}}
{{- end -}}
{{- end -}}

{{/*
//////////////////////////////////////////////////////////////////////////////////////////// 
*/}}

{{- define "perforator.agent.service.name" -}}
{{ printf "%s-agent-service" (include "perforator.fullname" .) }}
{{- end }}

{{/*
//////////////////////////////////////////////////////////////////////////////////////////// 
*/}}

{{- define "perforator.storage.host" -}}
{{- $hostNameOverride := coalesce .Values.agent.config.storageHostnameOverride .Values.storage.hostname  -}}
{{ $hostNameOverride | default (printf "%s:%v" (include "perforator.storage.service.name" .) .Values.storage.service.ports.grpc.port) }}
{{- end }}

{{- define "perforator.storage.service.name" -}}
{{ printf "%s-storage-service" (include "perforator.fullname" .) }}
{{- end }}

{{/*
//////////////////////////////////////////////////////////////////////////////////////////// 
*/}}

{{- define "perforator.proxy.host.http" -}}
{{- $hostNameOverride := coalesce .Values.web.config.HTTPProxyHostnameOverride .Values.proxy.hostname  -}}
{{ $hostNameOverride | default (printf "%s:%v" (include "perforator.proxy.service.name" .) .Values.proxy.service.ports.http.port) }}
{{- end }}

{{- define "perforator.proxy.host.grpc" -}}
{{- $hostNameOverride := coalesce .Values.web.config.GRPCProxyHostnameOverride .Values.proxy.hostname  -}}
{{ $hostNameOverride | default (printf "%s:%v" (include "perforator.proxy.service.name" .) .Values.proxy.service.ports.grpc.port) }}
{{- end }}

{{- define "perforator.proxy.service.name" -}}
{{ printf "%s-proxy-service" (include "perforator.fullname" .) }}
{{- end }}

{{/*
Construct appropriate URL prefix for task results
*/}}
{{- define "perforator.proxy.url_prefix" -}}
{{/* Check if both values are set - which is not allowed */}}
{{- if and .Values.proxy.url_prefix .Values.web.host -}}
    {{- fail "Error: Only one of proxy.url_prefix or web.host should be specified, not both. Use web.host when web service is enabled, or proxy.url_prefix for direct S3 access when web is disabled." -}}
{{- end -}}

{{/* If web.host is set, construct the URL with the standard path */}}
{{- if .Values.web.host -}}
    {{- $host := .Values.web.host -}}
    {{- if hasSuffix "/" $host -}}
        {{- $host = trimSuffix "/" $host -}}
    {{- end -}}
    {{- printf "%s/static/results/" $host -}}
{{/* If proxy.url_prefix is set, use it directly */}}
{{- else if .Values.proxy.url_prefix -}}
    {{- .Values.proxy.url_prefix -}}
{{/* If neither is set, fail with error message */}}
{{- else if .Values.proxy.enabled -}}
    {{- fail "Error: Either proxy.url_prefix or web.host must be specified. Use web.host when web service is enabled, or proxy.url_prefix for direct S3 access when web is disabled." -}}
{{- end -}}
{{- end -}}

{{/*
//////////////////////////////////////////////////////////////////////////////////////////// 
*/}}

{{- define "perforator.web.service.name" -}}
{{ printf "%s-web-service" (include "perforator.fullname" .) }}
{{- end }}

{{/*
//////////////////////////////////////////////////////////////////////////////////////////// 
*/}}

{{/*
//////////////////////////////////////////////////////////////////////////////////////////// 
*/}}

{{- define "perforator.gc.service.name" -}}
{{ printf "%s-gc-service" (include "perforator.fullname" .) }}
{{- end }}

{{/*
//////////////////////////////////////////////////////////////////////////////////////////// 
*/}}

{{- define "perforator.ingress.apiVersion" -}}
  {{- if semverCompare "<1.14-0" .Capabilities.KubeVersion.GitVersion -}}
    {{- print "extensions/v1beta1" -}}
  {{- else if semverCompare "<1.19-0" .Capabilities.KubeVersion.GitVersion -}}
    {{- print "networking.k8s.io/v1beta1" -}}
  {{- else -}}
    {{- print "networking.k8s.io/v1" -}}
  {{- end -}}
{{- end -}}
