Note: don't trust anything in this doc yet! :) still a work in progress

# User-Assigned Pod Identity

This assumes you have already created an Azure Identity, perhaps by using `az identity create`.

Define "Contributor" roles for your resource group for each user-assigned identity in your cluster so that the AzureIdentity can be bound to your VM/VMSS.
The controller runs on the controlplane/master nodes so specifically the controlplane identity.

`az role assignment create --role "Contributor" --assignee <principalId> --scope <resource group>` 

Define "Managed Identity Operator" roles for each user assigned identity.

`az role assignment create --role "Managed Identity Operator" --assignee <principalId> --scope <resource ID of managed identity>`


You will need to create the Kubernetes resource for AzureIdentity, AzureIdentityBinding, which will allow for the creation of AzureAssignedIdentity resources.

Create an AzureIdentity configuration file for each user-assigned identity.

```
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentity
metadata:
    name: <identity-name> 
spec:
    type: 0
    ResourceID: /subscriptions/<sub-id>/resourcegroups/<resource-group>/providers/Microsoft.ManagedIdentity/userAssignedIdentities/<identity-name>
    ClientID: <identity-client-id> 
```

Create an AzureIdentityBinding configuration file for AzureIdentity that you created. User selector 'node-label-operator'.

```
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentityBinding
metadata:
    name: <name> 
spec:
    AzureIdentity: "<identity-name>"
    Selector: "node-label-operator"
```


If you edited `config/manager/manager.yaml`, make sure that pods still have the label 'aadpodidentity=node-label-operator'.
