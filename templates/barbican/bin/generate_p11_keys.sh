#!/bin/bash
# Copyright 2024.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
set -xe

{{- if and (index . "P11Enabled") .P11Enabled }}
echo "Creating  MKEK label {{ .P11MKEKLabel }}"
barbican-manage hsm check_mkek --label {{ .P11MKEKLabel }} || barbican-manage hsm gen_mkek --label {{ .P11MKEKLabel }}

echo "Creating  HMAC label {{ .P11HMACLabel }}"
barbican-manage hsm check_hmac --label {{ .P11HMACLabel }} || barbican-manage hsm gen_hmac --label {{ .P11HMACLabel }}
{{- end }}
