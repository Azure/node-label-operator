# ARM Tag to Node Label Synchronization

Based off of a pre-existing document.

## Purpose

The purpose of this Kubernetes controller is to sync ARM VM/VMSS tags and node labels in an AKS cluster.
Users can choose whether to only sync ARM tags as node labels, sync node labels as ARM tags,
or perform a two-way sync.

## Motivation

Multiple customers have required this synchronization.
Their motivation is billing organization, housekeeping and overall resource tracking which works well on ARM tags.

## How it will work

### Kubernetes Configuration

- Default settings will have one way synchronization with VM/VMSS tags as node labels.

- The controller can be run with one of the following authentication methods:
    - Service Principals.
    - User Assigned Identity via "Pod Identity".
- Configurations can be specified in a Kubernetes ConfigMap. Configurable options include:
    - `syncDirection`: Direction of synchronization. Default is `arm-to-node`. Other options are `two-way` and `node-to-arm`. <!--    - `interval`: Configurable interval for synchronization. -->
    - `labelPrefix`: The node label prefix, with a default of `azure.tags`. An empty prefix will be permitted. <!-- - `tagPrefix`: The ARM tag prefix (for node-to-ARM and two-way sync), with a default of `k8s.labels`. An empty prefix will be permitted. -->
    - `resourceGroupFilter`: The controller can be limited to run on only nodes within a resource group filter (i.e. nodes that exist in RG1, RG2, RG3). Default is `none` for no filter. Otherwise, use name of resource group.
    - `conflictPolicy`: The policy for conflicting tag/label values. ARM tags or node labels can be given priority. ARM tags have priority by default (`arm-precedence`). Another option is to not update tags and raise Kubernetes event (`ignore`) and `node-precedence`. 
- The controller runs as a deployment with 2 replicas. Leader election is enabled.
- A minimum sync period can be set in config/manager/manager.yaml. Give time as string with integer and unit suffixes ns, us, ms, s, m, or h (ex: "2h30m", "100ns"). Default is 10 hours, as in kubebuilder.
- Finished project will have sample YAML files for deployment, the options configmap, and managed identity will be provided with instructions on what to edit before applying to a cluster.

Sample configuration for options ConfigMap (let's call it `options-configmap.yaml`):

``` yaml
apiVersion: v1
kind: ConfigMap
metadata:
    name: tag-label-sync
    namespace: default
data:
    syncDirection: "arm-to-node"
    labelPrefix: "azure.tags"
    conflictPolicy: "arm-precedence"
    resourceGroupFilter: "none"
```

Sample configuration for authorization:

``` yaml
apiVersion: "aadpodidentity.k8s.io/v1"
    kind: AzureIdentity
    metadata:
        name: <a-idname> 
    spec:
        type: 0
        ResourceID: /subscriptions/<subid>/resourcegroups/<resource-group>/providers/Microsoft.ManagedIdentity/userAssignedIdentities/<name>
        ClientID: <clientId>
```
Set `type: 0` for user-assigned MSI or `type: 1` for Service Principal.

Sample configuration for config/manager/manager.yaml:
``` yaml
apiVersion: v1
kind: Namespace
metadata:
    labels:
        control-plane: controller-manager
    name: system
---
apiVersion: apps/v1
kind: Deployment
metadata:
    name: controller-manager
    namespace: system
    labels:
        control-plane: controller-manager
spec:
    selector:
        matchLabels:
            control-plane: controller-manager
        replicas: 2
        template:
            metadata:
                labels:
                    control-plane: controller-manager
            spec:
                containers:
                - command:
                    - /manager
                    args:
                    - --enable-leader-election
                    - --sync-period 10h
                    image: controller:latest
                    name: manager
                    resources:
                        limits:
                            cpu: 100m
                            memory: 30Mi
                    requests:
                        cpu: 100m
                        memory: 20Mi
            terminationGracePeriodSeconds: 10
```

After creating and/or editing configuration files, apply config map to cluster.

```kubectl apply -f options-configmap.yaml```

Apply identity (more on this later, but for now look at the [Azure/aad-pod-identity repository](https://github.com/Azure/aad-pod-identity)).

Compile and run the controller by running `make` and then `make run`.

### Pseudo Code

For each VM/VMSS and node:
- For any tag that exists on the VM/VMSS but does not exist as a label on the node, the label will be created, (and vice versa with labels and tags, if two-way sync is enabled).
- If there is a conflict where a tag and label exist with the same name and a different value,
      the default action is that nothing will be done to resolve the conflict and the conflict will raise a Kubernetes
      event.
- ARM tags will be added as node labels with configurable prefix, and a default prefix of `azure.tags`, with the form 
    `azure.tags/<tag-name>:<tag-value>`. This default prefix is to encourage the use of a prefix.
- Node tags may not follow Azure tag name conventions (such as "kubernetes.io/os=linux" which contains '/'),
    so in that case... TBD

## Implementation Challenges

- Currently, we need to wait for nodes to be ready to be able to run the controller and access VM/VMSS tags. This is not ideal.
- Cluster updates should not delete tags and labels.
- Differences in tag and label limitations. A max tag limit exists (50 on most Azure resources). Also, different character and string length restrictions. Modifications to either tags or labels to fit the other standard must be consistent so that the controller will recognize when a tagor label has already been added.

## Possible Extensions

- Annotations
- Taints

## Questions

- What is meant by a resource group filter? Won't the controller be run in a cluster with resources within a single resource group anyway?
- What kind of rules should be in place for conflicting tags/labels and strings that don't match naming rules when converted to a tag/label?
