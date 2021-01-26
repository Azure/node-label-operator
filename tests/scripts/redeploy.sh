#!/bin/bash

# exit if unsuccessful at any step
set -e
set -o pipefail

# delete deployment
kustomize build config/default | kubectl delete -f -

# push new image and redeploy
make docker-build docker-push
make deploy
kubectl apply -f config/samples/configmap.yaml
