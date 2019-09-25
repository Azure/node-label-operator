#!/bin/bash

# exit if unsuccessful at any step
set -e
set -o pipefail

NAME=node-label-aks-engine
RESOURCE_GROUP=${NAME}-rg
AZURE_AUTH_LOCATION=${PWD}/tests/aks-engine/creds.json
AZURE_IDENTITY_LOCATION=${PWD}/tests/aks-engine/identity.json

az group create --name $RESOURCE_GROUP --location westus2 

az ad sp create-for-rbac --role="Contributor" --scopes="/subscriptions/${AZURE_SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}" > $AZURE_AUTH_LOCATION
if [ $? -eq 0 ]; then
    echo "Created Contributor role for resource group ${RESOURCE_GROUP}"
else
    echo "Creating Contributor role for resource group ${RESOURCE_GROUP} failed"
fi

# deploy aks-engine cluster

AKS_ENGINE_CLIENT_ID=$(cat ${AZURE_AUTH_LOCATION} | jq -r .appId)
AKS_ENGINE_CLIENT_SECRET=$(cat ${AZURE_AUTH_LOCATION} | jq -r .password)

if [ -d "${PWD}/tests/aks-engine/_output/${NAME}-cluster" ]; then
    rm -rf ${PWD}/tests/aks-engine/_output/${NAME}-cluster
fi
aks-engine deploy --subscription-id $AZURE_SUBSCRIPTION_ID \
    --resource-group $RESOURCE_GROUP \
    --location westus2 \
    --dns-prefix ${NAME}-cluster \
    --api-model tests/aks-engine/kubernetes.json \
    --output-directory "${PWD}/tests/aks-engine/_output/${NAME}-cluster" \
    --client-id $AKS_ENGINE_CLIENT_ID \
    --client-secret $AKS_ENGINE_CLIENT_SECRET \
    --set servicePrincipalProfile.clientId="${AKS_ENGINE_CLIENT_ID}" \
    --set servicePrincipalProfile.secret="${AKS_ENGINE_CLIENT_SECRET}"

export KUBECONFIG="${PWD}/tests/aks-engine/_output/${NAME}-cluster/kubeconfig/kubeconfig.westus2.json"

# create MSI

az identity create -g $RESOURCE_GROUP -n ${NAME}-identity -o json > $AZURE_IDENTITY_LOCATION
if [ $? -eq 0 ]; then
    echo "Created identity for resource group ${RESOURCE_GROUP}, stored in ${AZURE_IDENTITY_LOCATION}"
else
    echo "Creating identity for resource group ${RESOURCE_GROUP} failed"
fi

RESOURCE_ID=$(cat ${AZURE_IDENTITY_LOCATION} | jq -r .id)
CLIENT_ID=$(cat ${AZURE_IDENTITY_LOCATION} | jq -r .clientId)
PRINCIPAL_ID=$(cat ${AZURE_IDENTITY_LOCATION} | jq -r .principalId)

# create roles
az role assignment create --role "Managed Identity Operator" --assignee $PRINCIPAL_ID --scope $RESOURCE_ID 
az role assignment create --role "Contributor" --assignee $PRINCIPAL_ID --scope /subscriptions/${AZURE_SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}

# create aadpodidentity.yaml in order to create AzureIdentity
sed 's/<subid>/'"${AZURE_SUBSCRIPTION_ID}"'/g' samples/aadpodidentity.yaml | \
    sed 's/<resource-group>/'"${RESOURCE_GROUP}"'/g' | \
    sed 's/<a-idname>/'"${NAME}"'-identity/g' | \
    sed 's/<name>/'"${NAME}"'-identity/g' | \
    sed 's/<clientId>/'"${CLIENT_ID}"'/g' \
    > tests/aks-engine/aadpodidentity.yaml
if [ $? -eq 0 ]; then
    echo "Generated aadpodidentity.yaml file"
else
    echo "Failed to generate aadpodidentity.yaml file"
fi


# create aadpodidentitybinding.yaml in order to create AzureIdentityBinding
sed 's/<binding-name>/'"${NAME}"'-identity-binding/g' samples/aadpodidentitybinding.yaml | \
    sed 's/<identity-name>/'"${NAME}"'-identity/g' | \
    sed 's/<selector-name>/node-label-operator/g' \
    > tests/aks-engine/aadpodidentitybinding.yaml
if [ $? -eq 0 ]; then
    echo "Generated aadpodidentitybinding.yaml file"
else
    echo "Failed to generate aadpodidentitybinding.yaml file"
fi

# apply aad pod identity stuff 
kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml
kubectl apply -f tests/aks-engine/aadpodidentity.yaml
kubectl apply -f tests/aks-engine/aadpodidentitybinding.yaml

# deploy controller 
make docker-build docker-push
make deploy
kubectl apply -f samples/configmap.yaml
