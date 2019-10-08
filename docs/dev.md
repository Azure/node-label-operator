## Developer Instructions

### Running locally

If using Azure AD Application ID and Secret credentials, set the following environment variables:

```sh
export AZURE_SUBSCRIPTION_ID=
export AZURE_TENANT_ID=
export AZURE_CLIENT_ID=
export AZURE_CLIENT_SECRET=
```

To compile the codebase, run `make`.
To run the controller locally, run `make run`.

### Running in cluster

Make sure you have authentication with aad-pod-identity set up. You will need to have a Dockerhub account and be logged in.

The file `config/default/manager_image_patch.yaml` should have `image: <your-image-name>` as a result of `docker-build`.

```sh
export IMG="<dockerhub-username>/node-label"
make docker-build docker-push
make deploy
```

### Testing

#### Unit tests

To run unit tests: `make test`.

#### End-to-end tests

To run end-to-end tests:

Create a cluster with the controller installed.

Set environment variables to work with service principal authentication, if not set up already.

```sh
export AZURE_SUBSCRIPTION_ID=
export AZURE_TENANT_ID=
export AZURE_CLIENT_ID=
export AZURE_CLIENT_SECRET=
```

Set `KUBECONFIG_OUT` to the contents of your kubeconfig file. This was done to make testing with Github Actions simpler.

```sh
export KUBECONFIG_OUT=$(<$KUBECONFIG)
```

Run `make e2e-test`.

#### Linting

Run `make lint` and correct any problems that show up.

#### Code coverage

You can get a better look at code coverage by using `go tool cover -html=cover.out`.
