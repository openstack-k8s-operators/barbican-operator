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

{{- if and (index . "PKCS11Enabled") .PKCS11Enabled }}

mkek_label=$(crudini --get /etc/barbican/barbican.conf.d/01-custom.conf p11_crypto_plugin mkek_label)
echo "Creating  MKEK label $mkek_label"
barbican-manage hsm check_mkek --label $mkek_label || barbican-manage hsm gen_mkek --label $mkek_label

hmac_label=$(crudini --get /etc/barbican/barbican.conf.d/01-custom.conf p11_crypto_plugin hmac_label)
echo "Creating  HMAC label $hmac_label"
barbican-manage hsm check_hmac --label $hmac_label || barbican-manage hsm gen_hmac --label $hmac_label
{{- end }}
