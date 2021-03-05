#!/usr/bin/env bash
#SECRET_FILE_PATH="/home/file/87lENOv"
USER_NAMESPACE="injection"
WEBHOOK_NAMESPACE="sidecar-injector"

cp -r deployment deploy

kubectl create ns "${WEBHOOK_NAMESPACE}"
kubectl create ns "${USER_NAMESPACE}"
#kubectl create secret generic bitfusion-secret -n "${USER_NAMESPACE}" --from-file="${SECRET_FILE_PATH}"

./deploy/webhook-create-signed-cert.sh \
    --service sidecar-injector-webhook-svc \
    --secret sidecar-injector-webhook-certs \
    --namespace "${WEBHOOK_NAMESPACE}"

cat deploy/mutatingwebhook.yaml | \
    deploy/webhook-patch-ca-bundle.sh > \
    deploy/mutatingwebhook-ca-bundle.yaml

cat deploy/validationwebhook.yaml | \
    deploy/webhook-patch-ca-bundle.sh > \
    deploy/validationwebhook-ca-bundle.yaml


kubectl create -f deploy/configmap.yaml
kubectl create -f deploy/deployment.yaml
kubectl create -f deploy/service.yaml
kubectl create -f deploy/mutatingwebhook-ca-bundle.yaml
kubectl create -f deploy/validationwebhook-ca-bundle.yaml


kubectl -n sidecar-injector get pod
