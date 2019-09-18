#!/bin/bash

az role assignment create --role "Managed Identity Operator" --assignee $SP_ID --scope /subscriptions/$AZURE_SUBSCRIPTION_ID/resourcegroups/$AZURE_RESOURCE_GROUP/providers/Microsoft.ManagedIdentity/userAssignedIdentities/$IDENTITY
