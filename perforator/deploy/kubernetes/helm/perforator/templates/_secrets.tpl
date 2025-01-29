{{/*
Tries to find a secret with a given name and use its value, if secret doesn't exist creates a new one with a default value.

Usage:
{{ include "perforator.secrets.lookup" (dict "nameSpace" "someNameSpace" "secretName" "someName" "key" "someKey" "defaultVal" "someDefaultVal") }}

*/}}
{{- define "perforator.secrets.lookup" -}}
{{- $secret := (lookup "v1" "Secret" .nameSpace .secretName) -}}
{{- $secretData := $secret.data -}}
{{- $val := "" -}}
{{- if and $secretData (hasKey $secretData .key) -}}
  {{- $val = index $secretData .key -}}
{{- else if .defaultVal -}}
  {{- $val = .defaultVal | b64enc | quote -}}
{{- end -}}
{{- if $val -}}
{{- printf "%s" $val -}}
{{- end -}}
{{- end -}}
