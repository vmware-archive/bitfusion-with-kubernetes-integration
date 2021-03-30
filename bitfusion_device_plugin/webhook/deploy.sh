#!/usr/bin/env bash

# Create namespace
WEBHOOK_NAMESPACE="bwki"
kubectl create ns "${WEBHOOK_NAMESPACE}"

CRTDIR=$(pwd)
echo $CRTDIR
CRTDIR=$CRTDIR"/webhook"
echo $CRTDIR

if [ -d $CRTDIR"/deploy" ]; then
    kubectl delete -f $CRTDIR/deploy/deploy_bitfusion_injector.yaml
    kubectl delete -f $CRTDIR/deploy/bitfusion_injector_service.yaml
    kubectl delete -f $CRTDIR/deploy/deploy_bitfusion_injector_webhook_configmap.yaml
    kubectl delete -f $CRTDIR/deploy/bitfusion_mutating_webhook_configuration.yaml
    kubectl delete -f $CRTDIR/deploy/bitfusion_service_account.yaml
    kubectl delete -f $CRTDIR/deploy/bitfusion_validating_webhook_configuration.yaml
fi

# Copy deployment
rm -fr $CRTDIR/deploy
cp -r $CRTDIR/deployment $CRTDIR/deploy

# Add permissions
chmod 777  $CRTDIR//deploy/webhook-create-signed-cert.sh
chmod 777  $CRTDIR//deploy/webhook-patch-ca-bundle.sh

# Create signed cert
$CRTDIR/deploy/webhook-create-signed-cert.sh \
    --service bwki-webhook-svc \
    --secret bwki-webhook-certs \
    --namespace "${WEBHOOK_NAMESPACE}"

cat $CRTDIR/deploy/bitfusion_mutating_webhook_configuration.yaml | \
    $CRTDIR/deploy/webhook-patch-ca-bundle.sh > \
    $CRTDIR/deploy/mutatingwebhook-ca-bundle.yaml

cat $CRTDIR/deploy/bitfusion_validating_webhook_configuration.yaml | \
    $CRTDIR/deploy/webhook-patch-ca-bundle.sh > \
    $CRTDIR/deploy/validationwebhook-ca-bundle.yaml


kubectl apply -f $CRTDIR/deploy/deploy_bitfusion_injector.yaml
kubectl apply -f $CRTDIR/deploy/bitfusion_injector_service.yaml
kubectl apply -f $CRTDIR/deploy/deploy_bitfusion_injector_webhook_configmap.yaml
kubectl apply -f $CRTDIR/deploy/bitfusion_service_account.yaml
kubectl apply -f $CRTDIR/deploy/validationwebhook-ca-bundle.yaml
kubectl apply -f $CRTDIR/deploy/mutatingwebhook-ca-bundle.yaml




kubectl -n bwki get pod
