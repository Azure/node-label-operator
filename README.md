# Node Label Operator

![](https://github.com/Azure/node-label-operator/workflows/CI/badge.svg) ![](https://github.com/Azure/node-label-operator/workflows/E2E/badge.svg)

## Overview

The purpose of this Kubernetes controller is to sync ARM VM/VMSS tags and node labels in an AKS cluster.

## Installation

### Prerequisites
- [Go >= 1.13](https://golang.org/dl/)
- [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli?view=azure-cli-latest)
- access to an Azure account

## Getting Started 

1. Create a cluster.

Create a cluster using either [AKS](https://docs.microsoft.com/en-us/azure/aks/kubernetes-walkthrough) or [aks-engine](https://github.com/Azure/aks-engine), if you don't already have one ready to go.

2. Create a managed service identity if you don't have one. If you have an AKS cluster, then you will use the MC\_ resource group.

```sh
export AZURE_RESOURCE_GROUP=<resource-group>
export AZURE_IDENTITY_LOCATION=~/identity.json
export AZURE_IDENTITY=<identity-name>

az identity create -g $AZURE_RESOURCE_GROUP -n ${AZURE_IDENTITY} -o json > $AZURE_IDENTITY_LOCATION

export AZURE_IDENTITY_RESOURCE_ID=$(cat ${AZURE_IDENTITY_LOCATION} | jq -r .id)
export AZURE_IDENTITY_CLIENT_ID=$(cat ${AZURE_IDENTITY_LOCATION} | jq -r .clientId)
export AZURE_IDENTITY_PRINCIPAL_ID=$(cat ${AZURE_IDENTITY_LOCATION} | jq -r .principalId)
```

3. Create roles for identity.

```sh
az role assignment create --role "Managed Identity Operator" --assignee $AZURE_IDENTITY_PRINCIPAL_ID --scope $AZURE_IDENTITY_RESOURCE_ID
az role assignment create --role "Contributor" --assignee $AZURE_IDENTITY_PRINCIPAL_ID --scope /subscriptions/${AZURE_SUBSCRIPTION_ID}/resourceGroups/${AZURE_RESOURCE_GROUP}
```

4. Create k8s resources, including controller deployment. 

Clone this repository. From the root directory of the repository, run:

```sh
cat config/quickstart/quickstart.yaml | envsubst | kubectl apply -f - 
```

To see the tags on your VM or VMSS synced as labels on nodes: `kubectl get nodes --show-labels`.

## Other Pages

- [Full Tutorial](https://github.com/Azure/node-label-operator/blob/master/docs/tutorial.md)
- [Setting up aad-pod-identity with user-assigned MSI](https://github.com/Azure/node-label-operator/blob/master/docs/aadpodidentity.md)
- [Debugging tips](https://github.com/Azure/node-label-operator/blob/master/docs/debugging.md)
- [Developer instructions](https://github.com/Azure/node-label-operator/blob/master/docs/dev.md)

## Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.opensource.microsoft.com.

When you submit a pull request, a CLA bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., status check, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.
