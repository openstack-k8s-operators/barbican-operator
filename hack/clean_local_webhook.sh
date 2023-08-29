#!/bin/bash
set -ex

oc delete validatingwebhookconfiguration/vbarbican.kb.io --ignore-not-found
oc delete mutatingwebhookconfiguration/mbarbican.kb.io --ignore-not-found
