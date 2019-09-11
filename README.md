
# Node Label Operator

## Overview

## Installation

## Setup Instructions

1. Create a cluster.
2. Authentication:
    1. If using Azure MSI (Managed Service Identity) through aad-pod-identity, create AzureIdentity and AzureIdentityBinding resources for your cluster, using the service principal or user-assigned identities already in your cluster. If using genesys, the user-assigned identity is created automatically. You may need to create a Managed Identity Operator RBAC role. `kubectl apply -f samples/aadpodidentity.yaml` and `kubectl apply -f samples/aadpodidentitybinding.yaml`.
    2. If using Azure AD Application ID and Secret credentials, set the following environment variables...
2. Set up the Kubernetes ConfigMap. `kubectl apply -f samples/configmap.yaml`. If you don't, default settings will be used.
    1. `syncDirection`
    2. `labelPrefix`
    3. `tagPrefix`
    4. `conflictPolicy`
    5. `resourceGroupFilter`
3. You can edit `config/manager/manager.yaml`. `sync-period`.
4. Run `make` to build the controller, then `make run` to run the controller on your cluster.

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
