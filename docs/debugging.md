## Debugging

### Logs

To view controller pods:
```
kubectl get pods --namespace=node-label-operator-system
```

To view logs for controller:
```
kubectl logs node-label-operator-controller-manager-##########-##### --namespace=node-label-operator-system --all-containers
```
There are two pods so check both if the first one does not seem to have helpful output.

### Options ConfigMap

The ConfigMap needs to be in the right namespace (node-label-operator-system) and named correctly (node-label-operator).

Do not set minSyncPeriod to too short a period since that may cause throttling. Kubernetes node resources emit many events so operator reconciliation can happen too often and make too many requests to Azure resources. You can always change the minSyncPeriod by editing the config map associated with the controller (`kubectl edit cm node-label-operator --namespace=node-label-operator`).

### Service Principal Authentication

For authentication with service principal, which is used to run the controlle locally, make sure all necessary environment variables are set to the service principal being used with your cluster. You need `AZURE_SUBSCRIPTION_ID`, `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, and `AZURE_CLIENT_SECRET`.

### AAD Pod Identity Authentication 

Check that you created the proper roles. If you're not sure, looking at the mic pod logs might help. Make sure to edit mic deployment (`kubectl edit deployment mic`) to have argument '--v=6' and print logs using `kubectl logs <mic-pod-name>`. Check all mic pods if you don't find the leader right away.

Make sure your selector for your identity binding 'node-label-operator' and your controller pods have labels 'aadpodidbinding=node-label-operator'. You can check by running `kubectl get pods --namespace=node-label-operator-system --show-labels`.

If authentication works initially and then stops working, double check that you have only one user-assigned identity assigned to the VM or VMSS that the operator is running on. You can check by running `az vmss identity show -g <resource-group> -n <vmss-name>` or `az vm identity show -g <resource-group> -n <vm-name>` to show all of the the user-assigned identities on a VM or VMSS.
