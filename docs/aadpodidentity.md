Note: don't trust anything in this doc yet! :) still a work in progress

# User-Assigned Pod Identity

This assumes you have already created an Azure Identity, perhaps by using `az identity create`.

Define "Contributor" roles for your resource group for each user-assigned identity assigned to a VM/VMSS with master/controlplane nodes, which is where
the controller will run.

`az role assignment create --role "Contributor" --assignee <principalId> --scope <resource group>` 

Define "Managed Identity Operator" roles for each user assigned identity assigned to a VM/VMSS with master/controlplane nodes.

`az role assignment create --role "Managed Identity Operator" --assignee <principalId> --scope <resource ID of managed identity>`


You will need to create the Kubernetes resources for AzureIdentity and AzureIdentityBinding, which will allow for the creation of AzureAssignedIdentity resources.

Create an AzureIdentity configuration file for each user-assigned identity on your master/controlplane VM/VMSS. Use `type: 0` for user-assigned MSI instead of service principal.

```yaml
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentity
metadata:
    name: <identity-name> 
spec:
    type: 0
    ResourceID: /subscriptions/<sub-id>/resourcegroups/<resource-group>/providers/Microsoft.ManagedIdentity/userAssignedIdentities/<identity-name>
    ClientID: <identity-client-id> 
```

Then run `kubectl apply -f <aadpodidentity-config-file>.yaml`.

Create an AzureIdentityBinding configuration file for each AzureIdentity that you created. User selector 'node-label-operator'.

```yaml
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentityBinding
metadata:
    name: <name> 
spec:
    AzureIdentity: "<identity-name>"
    Selector: "node-label-operator"
```

Then run `kubectl apply -f <aadpodidentitybinding-config-file>.yaml`.


If you edited `config/manager/manager.yaml`, make sure that pods still have the label 'aadpodidbinding=node-label-operator'.

