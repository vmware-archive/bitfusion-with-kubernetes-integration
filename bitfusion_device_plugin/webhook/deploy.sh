#!/usr/bin/env bash

# Create namespace
WEBHOOK_NAMESPACE="bwki"
kubectl create ns "${WEBHOOK_NAMESPACE}"

CRTDIR=$(pwd)
echo $CRTDIR
CRTDIR=$CRTDIR"/webhook"
echo $CRTDIR

if [ -d $CRTDIR"/deploy" ]; then
    kubectl delete -f $CRTDIR/deploy/deploy-bitfusion-injector.yaml
    kubectl delete -f $CRTDIR/deploy/bitfusion-injector-service.yaml
    kubectl delete -f $CRTDIR/deploy/deploy-bitfusion-injector-webhook-configmap.yaml
    kubectl delete -f $CRTDIR/deploy/bitfusion-mutating-webhook-configuration.yaml
    kubectl delete -f $CRTDIR/deploy/bitfusion-service-account.yaml
    kubectl delete -f $CRTDIR/deploy/bitfusion_validating_webhook_configuration.yaml
    kubectl delete -f $CRTDIR/deploy/bitfusion-client-configmap.yaml
    kubectl delete -f $CRTDIR/deploy/bitfusion-client-info-configmap.yaml
fi

# Copy deployment
rm -fr $CRTDIR/deploy
cp -r $CRTDIR/deployment $CRTDIR/deploy

# Add permissions
chmod 777  $CRTDIR//deploy/webhook-create-signed-cert.sh
chmod 777  $CRTDIR//deploy/webhook-patch-ca-bundle.sh
chmod 777  $CRTDIR//deploy/webhook-create-ca.sh

# Create signed cert
echo "K8S_PLATFORM == ${K8S_PLATFORM}"

if [ "${K8S_PLATFORM}" == "tkgi" ]; then
    echo "Run webhook-create-ca.sh"
    $CRTDIR/deploy/webhook-create-ca.sh
else
    $CRTDIR/deploy/webhook-create-signed-cert.sh \
        --service bwki-webhook-svc \
        --secret bwki-webhook-certs \
        --namespace "${WEBHOOK_NAMESPACE}"
fi

cat $CRTDIR/deploy/bitfusion-mutating-webhook-configuration.yaml | \
    $CRTDIR/deploy/webhook-patch-ca-bundle.sh > \
    $CRTDIR/deploy/mutatingwebhook-ca-bundle.yaml

cat $CRTDIR/deploy/bitfusion-validating-webhook-configuration.yaml | \
    $CRTDIR/deploy/webhook-patch-ca-bundle.sh > \
    $CRTDIR/deploy/validationwebhook-ca-bundle.yaml


kubectl create -f $CRTDIR/deploy/deploy-bitfusion-injector.yaml
kubectl create -f $CRTDIR/deploy/bitfusion-injector-service.yaml
kubectl create -f $CRTDIR/deploy/deploy-bitfusion-injector-webhook-configmap.yaml
kubectl create -f $CRTDIR/deploy/bitfusion-service-account.yaml
kubectl create -f $CRTDIR/deploy/validationwebhook-ca-bundle.yaml
kubectl create -f $CRTDIR/deploy/mutatingwebhook-ca-bundle.yaml
kubectl create -f $CRTDIR/deploy/bitfusion-client-configmap.yaml
kubectl create -f $CRTDIR/deploy/bitfusion-client-info-configmap.yaml



kubectl -n bwki get pod
