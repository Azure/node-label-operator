## Tutorial

1. Create cluster

2. Authenticate with [AAD Pod Identity](https://github.com/Azure/aad-pod-identity).
    1. Create identity, if not already created. You can find the identitie(s) with `az vmss identity show -g <resource-group> -n <vmss-name>` or `az vm identity show -g <resource-group> -n <vm-name>`. There should only be one identity.
    `az identity create -g $MC_RESOURCE_GROUP -n ${AKS_NAME}-identity -o json > $AZURE_IDENTITY_LOCATION`.

    2. Assign roles.
    ```sh
    az role assignment create --role "Managed Identity Operator" --assignee $PRINCIPAL_ID --scope $RESOURCE_ID
    az role assignment create --role "Contributor" --assignee $PRINCIPAL_ID --scope /subscriptions/${E2E_SUBSCRIPTION_ID}/resourceGroups/${MC_RESOURCE_GROUP}
    ```

    3. Create necessary aad-pod-identity resources.

    For deploying with RBAC-enabled:

    `kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml`

    4. Create identity resources.

    Follow the format for aadpodidentity.yaml and aadpodidentitybinding.yaml, but with the name of your identity, its client ID, and the subscription and
    resource group that it falls under.

    aadpodidentity.yaml:
    ```yaml
    apiVersion: "aadpodidentity.k8s.io/v1"
    kind: AzureIdentity
    metadata:
        name: <identity-name> 
    spec:
        type: 0
        ResourceID: /subscriptions/<subid>/resourcegroups/<resource-group>/providers/Microsoft.ManagedIdentity/userAssignedIdentities/<identity-name>
        ClientID: <client-id>
    ```

    aadpodidentitybinding.yaml:
    ```yaml
    apiVersion: "aadpodidentity.k8s.io/v1"
    kind: AzureIdentityBinding
    metadata:
        name: <binding-name> 
    spec:
        AzureIdentity: "<identity-name>"
        Selector: "<selector-name>"
    ```

    ```sh
    kubectl apply -f aadpodidentity.yaml
    kubectl apply -f aadpodidentitybinding.yaml
    ```

    An AzureAssignedIdentity will be created for each controller pod.

3. Create config map

Set up the Kubernetes ConfigMap. It must be named 'node-label-operator' and have namespace 'node-label-operator-system' to allow to controller to
watch it in addition to nodes. If you don't, default settings will be used. You won't be able to create the configmap
until the namespace has been created.

configmap.yaml:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
    name: node-label-operator 
    namespace: node-label-operator-system 
data:
    labelPrefix: "azure.tags"
    tagPrefix: ""
    conflictPolicy: "arm-precedence"
    syncDirection: "arm-to-node"
    resourceGroup: "shoshanargwestus2"
    minSyncPeriod: "1m"
```

```sh
kubectl apply -f configmap.yaml
```

| setting | description | default |
| ------- | ----------- | ------- |
| `syncDirection` | Direction of synchronization. Default is `arm-to-node`. Other options are `two-way` and `node-to-arm`. Currently only `arm-to-node` is fully implemented and tested. | `arm-to-node` |
| `labelPrefix` | The node label prefix. An empty prefix will be permitted. However if you use an empty prefix, node labels will not be deleted when the corresponding ARM tag is deleted so using a non-empty prefix is strongly recommended. | `azure.tags` |
| `conflictPolicy` | The policy for conflicting tag/label values. ARM tags or node labels can be given priority. ARM tags have priority by default (`arm-precedence`). Another option is to not update tags and raise Kubernetes event (`ignore`) and `node-precedence`. If set to `node-precedence`, labels will not be deleted when the corresponding tags are deleted, even if `syncDirection` is set to `arm-to-node`. | `arm-precedence` |
| `resourceGroupFilter` | The controller can be limited to run on only nodes within a resource group filter (i.e. nodes that exist in RG1 but not RG2 or RG3). Default is `none` for no filter. Otherwise, use name of (single) resource group. | `none` |
| `minSyncPeriod` | The minimum interval between updates to a node, in a format accepted by golang time library for Duration. Decimal numbers followed by time unit suffix. Valid time units are "ns", "us", "ms", "s", "m", "h". Ex: "300ms", "1.5h", or "2h45m". It may take one default period (5m) for this to update. | "5m" |
| `tagPrefix` | Not supported currently. | |


4. You can edit [`config/manager/manager.yaml`](https://github.com/Azure/node-label-operator/blob/master/config/manager/manager.yaml). `sync-period` is the maximum time between calls to reconcile. The default is "10h".

5. Deploy controller

<!-- make docker-build docker-push -->
make deploy



<!-- 4. Running the operator: -->
<!--     1. To run the controller locally, run `make` to build the controller, then `make run` to run the controller on your cluster. -->
<!--     2. To deploy the controller in your cluster, make sure IMG is set (for example, "<dockerhub-username>/node-label-manager") and run `make docker-build docker-push` and `make deploy`. -->
<!--     ```sh -->
<!--     export IMG=<dockerhub-username>/node-label-manager -->
<!--     make docker-build docker-push -->
<!--     make deploy -->
<!--     ``` -->

For a general idea of how to set up a cluster from scratch with this operator installed, see the commands used for setting up test clusters with
[AKS](https://github.com/Azure/node-label-operator/blob/master/tests/aks/setup.sh) and [aks-engine](https://github.com/Azure/node-label-operator/blob/master/tests/aks-engine/setup.sh).

To see the tags on your VM or VMSS synced as labels on nodes: `kubectl get nodes --show-labels`.

