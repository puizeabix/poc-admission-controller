#!/bin/bash

echo "Creating certificates"
mkdir certs
openssl genrsa -out certs/tls.key 2048
openssl req -new -key certs/tls.key -out certs/tls.csr -subj "/CN=poc-admission-controller.default.svc"
openssl x509 -req -extfile <(printf "subjectAltName=DNS:poc-admission-controller.default.svc") -in certs/tls.csr -signkey certs/tls.key -out certs/tls.crt

echo "Creating Webhook Server TLS Secret"
kubectl create secret tls poc-admission-controller-tls \
    --cert "certs/tls.crt" \
    --key "certs/tls.key"

echo "Creating Webhook Server Deployment"
kubectl create -f manifest/deployment.yaml

echo "Creating K8s Webhooks"
ENCODED_CA=$(cat certs/tls.crt | base64 | tr -d '\n')
sed -e 's@${ENCODED_CA}@'"$ENCODED_CA"'@g' <"manifest/admissionregistration.yaml" | kubectl create -f -