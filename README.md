# Node Label Operator

![](https://github.com/Azure/node-label-operator/workflows/CI/badge.svg) ![](https://github.com/Azure/node-label-operator/workflows/E2E/badge.svg)

## Overview

The purpose of this Kubernetes controller is to sync ARM VM/VMSS tags and node labels in an AKS cluster.

## Installation

### Prerequisites
- [Go >= 1.13](https://golang.org/dl/)
- [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli?view=azure-cli-latest)
- access to an Azure account

## Setup Instructions

1. Create a cluster.
2. Authentication:
    1. If using Azure MSI (Managed Service Identity) through [AAD Pod Identity](https://github.com/Azure/aad-pod-identity), create AzureIdentity and AzureIdentityBinding resources for your cluster,
    using the service principal or user-assigned identities already in your cluster.
    You may need to create a Managed Identity Operator RBAC role. You will need to define AzureIdentity and AzureIdentityBinding in configuration files
    and run something like `kubectl apply -f samples/aadpodidentity.yaml` and `kubectl apply -f samples/aadpodidentitybinding.yaml` where
    [`samples/aadpodidentity.yaml`](https://github.com/Azure/node-label-operator/blob/master/samples/aadpodidentity.yaml) and
    [`samples/aadpodidentitybinding.yaml`](https://github.com/Azure/node-label-operator/blob/master/samples/aadpodidentitybinding.yaml) are configuration files filled with your MSI information,
    in the format in the linked YAML files.
    You will need to have only one user-assigned identity on the compute resource (VM or VMSS) that the operator is running on.
    2. If using Azure AD Application ID and Secret credentials, set the following environment variables:
        ```
        export AZURE_SUBSCRIPTION_ID=
        export AZURE_TENANT_ID=
        export AZURE_CLIENT_ID=
        export AZURE_CLIENT_SECRET=
        ```
2. Set up the Kubernetes ConfigMap. It must be named 'node-label-operator' and have namespace 'node-label-operator-system' to allow to controller to
watch it in addition to nodes. `kubectl apply -f samples/configmap.yaml`. If you don't, default settings will be used. You won't be able to create the configmap
until the namespace has been created.
    1. `syncDirection`: Direction of synchronization. Default is `arm-to-node`. Other options are `two-way` and `node-to-arm`. Currently only `arm-to-node` is fully
    implemented and tested.
    2. `labelPrefix`: The node label prefix, with a default of `azure.tags`. An empty prefix will be permitted. However if you use an empty prefix, node labels
    will not be deleted when the corresponding ARM tag is deleted so using a non-empty prefix is strongly recommended.
    3. `tagPrefix`: Not supported currently.
    4. `conflictPolicy`: The policy for conflicting tag/label values. ARM tags or node labels can be given priority. ARM tags have priority by default
    (`arm-precedence`). Another option is to not update tags and raise Kubernetes event (`ignore`) and `node-precedence`. If set to `node-precedence`, labels will
    not be deleted when the corresponding tags are deleted, even if `syncDirection` is set to `arm-to-node`.
    5. `resourceGroupFilter`: The controller can be limited to run on only nodes within a resource group filter (i.e. nodes that exist in RG1, RG2, RG3).
    Default is `none` for no filter. Otherwise, use name of (single) resource group.
    6. `minSyncPeriod`: The minimum interval between updates to a node, in a format accepted by golang time library for Duration. Decimal numbers followed by
    time unit suffix. Valid time units are "ns", "us", "ms", "s", "m", "h". Ex: "300ms", "1.5h", or "2h45m". It may take one default period (5m) for this
    to update.
3. You can edit [`config/manager/manager.yaml`](https://github.com/Azure/node-label-operator/blob/master/config/manager/manager.yaml). `sync-period` is the maximum time between calls to reconcile. The default is "10h".
4. Running the operator:
    1. To run the controller locally, run `make` to build the controller, then `make run` to run the controller on your cluster.
    2. To deploy the controller in your cluster, make sure IMG is set (for example, "<dockerhub-username>/node-label-manager") and run `make docker-build docker-push` and `make deploy`.

For a general idea of how to set up a cluster from scratch with this operator installed, see the commands used for setting up test clusters with
[AKS](https://github.com/Azure/node-label-operator/blob/master/tests/aks/setup.sh) and [aks-engine](https://github.com/Azure/node-label-operator/blob/master/tests/aks-engine/setup.sh).

## Other Pages

- [Setting up aad-pod-identity with user-assigned MSI](https://github.com/Azure/node-label-operator/blob/master/docs/aadpodidentity.md)
- [Debugging tips](https://github.com/Azure/node-label-operator/blob/master/docs/debugging.md)

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
