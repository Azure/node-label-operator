// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

// copied from genesys

package azure

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

const (
	// AzurePublicCloud azure public cloud for now
	AzurePublicCloud = "AzurePublicCloud"
)

type AuthContext interface {
	Cloud() string
	ClientID() string
	ClientSecret() string
	TenantID() string
}

// authContext contains the necessary parameters for authentication to Azure resources.
type authContext struct {
	AzureCloud        string `json:"-"`
	AzureClientID     string `json:"clientId"`
	AzureClientSecret string `json:"clientSecret"`
	AzureTenantID     string `json:"tenantId"`
}

// NewAuthContext new auth context
func NewAuthContext() AuthContext {
	ctx, _ := provideConfiguration()
	return ctx
}

func (ac *authContext) Cloud() string {
	return ac.AzureCloud
}

func (ac *authContext) ClientID() string {
	return ac.AzureClientID
}

func (ac *authContext) ClientSecret() string {
	return ac.AzureClientSecret
}

func (ac *authContext) TenantID() string {
	return ac.AzureTenantID
}

func injectAuthorizer() (autorest.Authorizer, error) {
	config, err := provideConfiguration()
	if err != nil {
		return nil, err
	}
	return provideAuthorizer(config)
}

func injectGraphAuthorizer() (autorest.Authorizer, error) {
	config, err := provideConfiguration()
	if err != nil {
		return nil, err
	}
	return provideGraphAuthorizer(config)
}

func provideConfiguration() (*authContext, error) {
	file, err := auth.GetSettingsFromFile() // comes from AZURE_AUTH_LOCATION
	if err != nil {
		env, err := auth.GetSettingsFromEnvironment() // comes from env variables
		if err != nil {
			return &authContext{}, err
		}
		config, err := env.GetClientCredentials()
		if err != nil {
			config, err = getMSICredentials() // do I need to have this be priority?
			if err != nil {
				return &authContext{}, err
			}
		}
		return &authContext{
			AzureClientID:     config.ClientID,
			AzureClientSecret: config.ClientSecret,
			AzureTenantID:     config.TenantID,
			AzureCloud:        env.Environment.Name,
		}, nil
	}
	return &authContext{
		AzureClientID:     file.Values[auth.ClientID],
		AzureClientSecret: file.Values[auth.ClientSecret],
		AzureTenantID:     file.Values[auth.TenantID],
		AzureCloud:        AzurePublicCloud,
	}, nil
}

func provideAuthorizer(ac *authContext) (autorest.Authorizer, error) {
	env, err := azure.EnvironmentFromName(ac.AzureCloud)
	if err != nil {
		return nil, err
	}
	return provideResourceAuthorizer(env.ResourceManagerEndpoint)
}

func provideGraphAuthorizer(ac *authContext) (autorest.Authorizer, error) {
	env, err := azure.EnvironmentFromName(ac.AzureCloud)
	if err != nil {
		return nil, err
	}
	return provideResourceAuthorizer(env.GraphEndpoint)
}

func provideResourceAuthorizer(resource string) (autorest.Authorizer, error) {
	authorizer, err := auth.NewAuthorizerFromFileWithResource(resource)
	if err != nil {
		return auth.NewAuthorizerFromEnvironmentWithResource(resource)
	}
	return authorizer, nil
}

func getMSICredentials() (auth.ClientCredentialsConfig, error) {
	r, err := requestIdentityToken()
	if err != nil {
		return auth.ClientCredentialsConfig{}, err
	}
	return auth.ClientCredentialsConfig{ClientID: r.ClientID, TenantID: ""}, nil
}

type responseToken struct {
	AccessToken  string `json:"access_token"`
	ClientID     string `json:"client_id"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    string `json:"expires_in"`
	ExpiresOn    string `json:"expires_on"`
	NotBefore    string `json:"not_before"`
	Resource     string `json:"resource"`
	TokenType    string `json:"token_type"`
}

func requestIdentityToken() (*responseToken, error) {
	endpoint, err := url.Parse("http://169.254.169.254/metadata/identity/oauth2/token")
	if err != nil {
		return nil, err
	}

	parameters := url.Values{}
	parameters.Add("resource", "https://management.azure.com/")
	parameters.Add("api-version", "2018-02-01")
	endpoint.RawQuery = parameters.Encode()
	req, err := http.NewRequest("GET", endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Metadata", "true")

	client := &http.Client{}
	client.Timeout = 5 * time.Second
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	responseBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	var r responseToken
	err = json.Unmarshal(responseBytes, &r)
	if err != nil {
		return nil, err
	}

	return &r, nil
}
