#!/bin//bash
#
# Copyright 2020 Red Hat Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License"); you may
# not use this file except in compliance with the License. You may obtain
# a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
# WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
# License for the specific language governing permissions and limitations
# under the License.
set -ex

# This script generates the barbican.conf file and
# copies the result to the ephemeral /var/lib/config-data/merged volume.

export PASSWORD=${BarbicanPassword:?"Please specify a BarbicanPassword variable."}
export DBHOST=${DatabaseHost:?"Please specify a DatabaseHost variable."}
export DBUSER=${DatabaseUser:-"barbican"}
export DB=${DatabaseName:-"barbican"}
export DBPASSWORD=${DatabasePassword:?"Please specify a DatabasePassword variable."}


DEFAULT_DIR=/var/lib/config-data/default
CUSTOM_DIR=/var/lib/config-data/custom
MERGED_DIR=/var/lib/config-data/merged
SVC_CFG=/etc/barbican/barbican.conf
SVC_CFG_MERGED=${MERGED_DIR}/barbican.conf
SVC_CFG_MERGED_DIR=${MERGED_DIR}/barbican.conf.d

mkdir -p ${SVC_CFG_MERGED_DIR}

cp ${DEFAULT_DIR}/* ${MERGED_DIR}

# Save the default service config from container image as barbican.conf.sample,
# and create a small barbican.conf file that directs people to files in
# barbican.conf.d.

cp -a ${SVC_CFG} ${SVC_CFG_MERGED}.sample
cat <<EOF > ${SVC_CFG_MERGED}
# Service configuration snippets are stored in the barbican.conf.d subdirectory.
EOF

cp ${DEFAULT_DIR}/barbican.conf ${SVC_CFG_MERGED_DIR}/00-default.conf

# Generate 01-deployment-secrets.conf
DEPLOYMENT_SECRETS=${SVC_CFG_MERGED_DIR}/01-deployment-secrets.conf

cat <<EOF >> ${DEPLOYMENT_SECRETS}
[DEFAULT]
sql_connection = mysql+pymysql://${DBUSER}:${DBPASSWORD}@${DBHOST}/${DB}

[keystone_authtoken]
password = ${PASSWORD}

EOF

if [ -f ${DEFAULT_DIR}/custom.conf ]; then
    cp ${DEFAULT_DIR}/custom.conf ${SVC_CFG_MERGED_DIR}/80-custom.conf
fi

SECRET_FILES="$(ls /var/lib/config-data/secret-*/* 2>/dev/null || true)"
if [ -n "${SECRET_FILES}" ]; then
    cat ${SECRET_FILES} > ${SVC_CFG_MERGED_DIR}/90-secrets.conf
fi

chown -R :barbican ${SVC_CFG_MERGED_DIR}
