# User-Assigned Pod Identity

This assumes you have already created an Azure Identity, perhaps by using `az identity create`.

Define "Virtual Machine Contributor" roles for each VM/VMSS according to its user-assigned identity in your cluster so that the AzureIdentity can be bound to your VM/VMSS.

`az role assignment create --role "Virtual Machine Contributor" --assignee <principalId> --scope <resource ID of VM or VMSS>` 

Define "Managed Identity Operator" roles for each user assigned identity.

`az role assignment create --role "Managed Identity Operator" --assignee <principalId> --scope <resource ID of managed identity>`


You will need to create the Kubernetes resource for AzureIdentity, AzureIdentityBinding, which will allow for the creation of AzureAssignedIdentity resources.

Create an AzureIdentity configuration file for each user-assigned identity.

Create an AzureIdentityBinding configuration file for AzureIdentity that you created. User selector 'node-label-operator'.


If you edited config/manager/manager.yaml, make sure that pods still have the label 'aadpodidentity=node-label-operator'.
