## Developer Instructions

### Running locally

If using Azure AD Application ID and Secret credentials, set the following environment variables:

```sh
export AZURE_SUBSCRIPTION_ID=
export AZURE_TENANT_ID=
export AZURE_CLIENT_ID=
export AZURE_CLIENT_SECRET=
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

Run `make e2e-test`.

