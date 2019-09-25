#!/bin/bash

NAME=node-label-aks
RESOURCE_GROUP=${NAME}-rg
MC_RESOURCE_GROUP=MC_${RESOURCE_GROUP}_${NAME}-cluster_westus2

# mkdir ~/aks 
AZURE_AUTH_LOCATION=${PWD}/tests/aks/creds.json
AZURE_IDENTITY_LOCATION=${PWD}/tests/aks/identity.json

az ad sp create-for-rbac --skip-assignment > $AZURE_AUTH_LOCATION

AKS_CLIENT_ID=$(cat ${AZURE_AUTH_LOCATION} | jq -r .appId)
AKS_CLIENT_SECRET=$(cat ${AZURE_AUTH_LOCATION} | jq -r .password)

az group create --name $RESOURCE_GROUP --location westus2

az aks create \
    --resource-group $RESOURCE_GROUP \
    --name ${NAME}-cluster \
    --node-count 5 \
    --service-principal $AKS_CLIENT_ID \
    --client-secret $AKS_CLIENT_SECRET \
    --generate-ssh-keys

az aks get-credentials --resource-group $RESOURCE_GROUP --name ${NAME}-cluster --location westus2

KUBECONFIG="$HOME/.kube/config"

az identity create -g $MC_RESOURCE_GROUP -n ${NAME}-identity -o json > $AZURE_IDENTITY_LOCATION

RESOURCE_ID=$(cat ${AZURE_IDENTITY_LOCATION} | jq -r .id)
CLIENT_ID=$(cat ${AZURE_IDENTITY_LOCATION} | jq -r .clientId)
PRINCIPAL_ID=$(cat ${AZURE_IDENTITY_LOCATION} | jq -r .principalId)

az role assignment create --role "Managed Identity Operator" --assignee $PRINCIPAL_ID --scope $RESOURCE_ID
az role assignment create --role "Contributor" --assignee $PRINCIPAL_ID --scope /subscriptions/${AZURE_SUBSCRIPTION_ID}/resourceGroups/$MC_RESOURCE_GROUP

# create aadpodidentity.yaml in order to create AzureIdentity
sed 's/<subid>/'"${AZURE_SUBSCRIPTION_ID}"'/g' samples/aadpodidentity.yaml | \
    sed 's/<resource-group>/'"${RESOURCE_GROUP}"'/g' | \
    sed 's/<a-idname>/'"${NAME}"'-identity/g' | \
    sed 's/<name>/'"${NAME}"'-identity/g' | \
    sed 's/<clientId>/'"${CLIENT_ID}"'/g' \
    > tests/aks/aadpodidentity.yaml
if [ $? -eq 0 ]; then
    echo "Generated aadpodidentity.yaml file"
else
    echo "Failed to generate aadpodidentity.yaml file"
fi

# create aadpodidentitybinding.yaml in order to create AzureIdentityBinding
sed 's/<binding-name>/'"${NAME}"'-identity-binding/g' samples/aadpodidentitybinding.yaml | \
    sed 's/<identity-name>/'"${NAME}"'-identity/g' \
    sed 's/<selector-name>/node-label-operator/g' \
    > tests/aks/aadpodidentitybinding.yaml
if [ $? -eq 0 ]; then
    echo "Generated aadpodidentitybinding.yaml file"
else
    echo "Failed to generate aadpodidentitybinding.yaml file"
fi

kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml
kubectl apply -f tests/config/aks-aadpodidentity.yaml
kubectl apply -f tests/config/aks-aadpodidentitybinding.yaml

make docker-build docker-push
make deploy
kubectl apply -f samples/configmap.yaml
