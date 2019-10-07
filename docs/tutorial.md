Tutorial

1. Create cluster

2. Authenticate with AAD Pod Identity 
  - create identity, if not already created
  - assign roles
  - deploy necessary stuff
  - create k8s resources

3. Create config map

| setting | description | default |
| ------- | ----------- | ------- |
| `syncDirection` | Direction of synchronization. Default is `arm-to-node`. Other options are `two-way` and `node-to-arm`. Currently only `arm-to-node` is fully implemented and tested. | `arm-to-node` |
| `labelPrefix` | The node label prefix. An empty prefix will be permitted. However if you use an empty prefix, node labels will not be deleted when the corresponding ARM tag is deleted so using a non-empty prefix is strongly recommended. | `azure.tags` |
| `conflictPolicy` | The policy for conflicting tag/label values. ARM tags or node labels can be given priority. ARM tags have priority by default (`arm-precedence`). Another option is to not update tags and raise Kubernetes event (`ignore`) and `node-precedence`. If set to `node-precedence`, labels will not be deleted when the corresponding tags are deleted, even if `syncDirection` is set to `arm-to-node`. | `arm-precedence` |
| `resourceGroupFilter` | The controller can be limited to run on only nodes within a resource group filter (i.e. nodes that exist in RG1 but not RG2 or RG3). Default is `none` for no filter. Otherwise, use name of (single) resource group. | `none` |
| `minSyncPeriod` | The minimum interval between updates to a node, in a format accepted by golang time library for Duration. Decimal numbers followed by time unit suffix. Valid time units are "ns", "us", "ms", "s", "m", "h". Ex: "300ms", "1.5h", or "2h45m". It may take one default period (5m) for this to update. | "5m" |


4. Deploy controller

<!-- make docker-build docker-push -->
make deploy
