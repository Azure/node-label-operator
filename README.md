
# Node Label Operator

## Overview

The purpose of this Kubernetes controller is to sync ARM VM/VMSS tags and node labels in an AKS cluster.

## Installation

### Prerequisites
- Go 1.12.9 and above
- Kubernetes
- Azure CLI
- access to an Azure account

## Setup Instructions

1. Create a cluster.
2. Authentication:
    1. If using Azure MSI (Managed Service Identity) through aad-pod-identity, create AzureIdentity and AzureIdentityBinding resources for your cluster, using the service principal or user-assigned identities already in your cluster. If using genesys, the user-assigned identity is created automatically. You may need to create a Managed Identity Operator RBAC role. `kubectl apply -f samples/aadpodidentity.yaml` and `kubectl apply -f samples/aadpodidentitybinding.yaml`.
    2. If using Azure AD Application ID and Secret credentials, set the following environment variables...
        ```
        export AZURE_SUBSCRIPTION_ID=
        export AZURE_TENANT_ID=
        export AZURE_CLIENT_ID=
        export AZURE_CLIENT_SECRET=
        ```
2. Set up the Kubernetes ConfigMap. It must be named 'node-label-controller' and have namespace 'node-label-controller-system' to allow to controller to watch it in addition to nodes. `kubectl apply -f samples/configmap.yaml`. If you don't, default settings will be used.
    1. `syncDirection`: Direction of synchronization. Default is `arm-to-node`. Other options are `two-way` and `node-to-arm`. 
    2. `labelPrefix`: The node label prefix, with a default of `azure.tags`. An empty prefix will be permitted.
    3. `tagPrefix`
    4. `conflictPolicy`: The policy for conflicting tag/label values. ARM tags or node labels can be given priority. ARM tags have priority by default (`arm-precedence`). Another option is to not update tags and raise Kubernetes event (`ignore`) and `node-precedence`.
    5. `resourceGroupFilter`: The controller can be limited to run on only nodes within a resource group filter (i.e. nodes that exist in RG1, RG2, RG3). Default is `none` for no filter. Otherwise, use name of (single) resource group.
3. You can edit `config/manager/manager.yaml`. `sync-period`.
4. To run the controller locally, run `make` to build the controller, then `make run` to run the controller on your cluster. To deploy the controller in your cluster, run `make docker-build docker-push` and `make deploy`.

## Other Pages (coming soon)

- Setting up aad-pod-identity with user-assigned MSI
- Debugging?

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
