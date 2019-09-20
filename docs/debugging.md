Note: not finished at all

## Debugging

- To view logs for controller: `kubectl logs node-label-operator-controller-manager-##########-##### --namespace=node-label-operator-system --all-containers`
- Make sure you set the IMG environment variable to docker image name and do `make docker-build docker-push` before running `make deploy`.
- The configmap needs to be in the right namespace (node-label-operator-system) and named correctly (node-label-operator).

### Service Principal Authentication

- For authentication with service principal, make sure all necessary environment variables are set to the service principal being used with your cluster. You need `AZURE_SUBSCRIPTION_ID`, `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, and `AZURE_CLIENT_SECRET`.

### aad-pod-identity Authentication 

- Check that you created the proper roles. If you're not sure, looking the mic pod logs might help. Make sure to edit mic deployment to have argument '--v=6' and print logs using `kubectl logs <mic-pod-name>`. Check all mic pods if you don't find the leader right away.
- Make sure your selector for your identity binding 'node-label-operator' and your controller pods have labels 'aadpodidbinding=node-label-operator'. You can check by running `kubectl get pods --namespace=node-label-operator-system --show-labels`.
