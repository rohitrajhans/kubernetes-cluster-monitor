#!/bin/bash

kubectl create -f inject_sidecar.yaml

./webhook-create-signed-cert.sh \
--service sidecar-injector-webhook-svc \
--secret sidecar-injector-webhook-certs \
--namespace sidecar-injector

kubectl create -f clusterrole.yaml
kubectl create -f deployment.yaml
kubectl create -f service.yaml

cat mutating-webhook.yaml | ./webhook-patch-ca-bundle.sh > mutatingwebhook-ca-bundle.yaml
kubectl create -f mutatingwebhook-ca-bundle.yaml

