#!/bin/bash

set -e
set -o pipefail

export AKS_NAME=node-label-aks
export AKS_RESOURCE_GROUP=${AKS_NAME}-rg
export MC_RESOURCE_GROUP=MC_${AKS_RESOURCE_GROUP}_${AKS_NAME}-cluster_westus2

# mkdir ~/aks 
export AZURE_AUTH_LOCATION=${PWD}/tests/aks/creds.json
export AZURE_IDENTITY_LOCATION=${PWD}/tests/aks/identity.json

az ad sp create-for-rbac --skip-assignment > $AZURE_AUTH_LOCATION

export AKS_CLIENT_ID=$(cat ${AZURE_AUTH_LOCATION} | jq -r .appId)
export AKS_CLIENT_SECRET=$(cat ${AZURE_AUTH_LOCATION} | jq -r .password)

az group create --name $AKS_RESOURCE_GROUP --location westus2

az aks create \
    --resource-group $AKS_RESOURCE_GROUP \
    --name ${AKS_NAME}-cluster \
    --node-count 3 \
    --service-principal $AKS_CLIENT_ID \
    --client-secret $AKS_CLIENT_SECRET \
    --generate-ssh-keys

az aks get-credentials --resource-group $AKS_RESOURCE_GROUP --name ${AKS_NAME}-cluster

export KUBECONFIG="$HOME/.kube/config"

az identity create -g $MC_RESOURCE_GROUP -n ${AKS_NAME}-identity -o json > $AZURE_IDENTITY_LOCATION
if [ $? -eq 0 ]; then
    echo "Created identity for resource group ${MC_RESOURCE_GROUP}, stored in ${AZURE_IDENTITY_LOCATION}"
else
    echo "Creating identity for resource group ${MC_RESOURCE_GROUP} failed"
fi

export RESOURCE_ID=$(cat ${AZURE_IDENTITY_LOCATION} | jq -r .id)
export CLIENT_ID=$(cat ${AZURE_IDENTITY_LOCATION} | jq -r .clientId)
export PRINCIPAL_ID=$(cat ${AZURE_IDENTITY_LOCATION} | jq -r .principalId)

az role assignment create --role "Managed Identity Operator" --assignee $PRINCIPAL_ID --scope $RESOURCE_ID
az role assignment create --role "Contributor" --assignee $PRINCIPAL_ID --scope /subscriptions/${AZURE_SUBSCRIPTION_ID}/resourceGroups/${MC_RESOURCE_GROUP}

# create aadpodidentity.yaml in order to create AzureIdentity
sed 's/<subid>/'"${AZURE_SUBSCRIPTION_ID}"'/g' samples/aadpodidentity.yaml | \
    sed 's/<resource-group>/'"${MC_RESOURCE_GROUP}"'/g' | \
    sed 's/<a-idname>/'"${AKS_NAME}"'-identity/g' | \
    sed 's/<name>/'"${AKS_NAME}"'-identity/g' | \
    sed 's/<clientId>/'"${CLIENT_ID}"'/g' \
    > tests/aks/aadpodidentity.yaml
if [ $? -eq 0 ]; then
    echo "Generated aadpodidentity.yaml file"
else
    echo "Failed to generate aadpodidentity.yaml file"
fi

# create aadpodidentitybinding.yaml in order to create AzureIdentityBinding
sed 's/<binding-name>/'"${AKS_NAME}"'-identity-binding/g' samples/aadpodidentitybinding.yaml | \
    sed 's/<identity-name>/'"${AKS_NAME}"'-identity/g' | \
    sed 's/<selector-name>/node-label-operator/g' \
    > tests/aks/aadpodidentitybinding.yaml
if [ $? -eq 0 ]; then
    echo "Generated aadpodidentitybinding.yaml file"
else
    echo "Failed to generate aadpodidentitybinding.yaml file"
fi

kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml
kubectl apply -f tests/aks/aadpodidentity.yaml
kubectl apply -f tests/aks/aadpodidentitybinding.yaml

make docker-build docker-push
make deploy
kubectl apply -f samples/configmap.yaml
