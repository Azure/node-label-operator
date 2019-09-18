## Debugging

- Did you remember to do `make docker-build docker-push` before running `make deploy`?
- Is your configmap in the right namespace (node-label-operator-system)and is it named correctly (node-label-operator)?
- If using aad-pod-identity, did you create the proper roles?
- If using aad-pod-identity, is your selector for your identity binding 'node-label-operator' and do your controller pods have labels 'aadpodidbinding=node-label-operator'?
