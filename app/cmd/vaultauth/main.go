package main

import (
	"context"
	"encoding/json"
	"fmt"
	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/approle"
	"io/ioutil"
	"os"
)

type vaultStorage struct{
	client *vault.Client
	appRole *auth.AppRoleAuth
	secretPath string
	secretKey string
}

func newClient() (*vault.Client, error){
	config := vault.DefaultConfig() // modify for more granular configuration
	tlsConfig := vault.TLSConfig{
		CACert:        "",
		CAPath:        "",
		ClientCert:    "",
		ClientKey:     "",
		TLSServerName: "",
		Insecure:      true,
	}
	config.ConfigureTLS(&tlsConfig)

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize Vault client: %w", err)
	}
	return client, nil
}


func newStorage(client *vault.Client, wrappenToken bool) *vaultStorage {
	roleID := os.Getenv("APPROLE_ROLE_ID")
	if roleID == "" {
		fmt.Println("no role ID was provided in APPROLE_ROLE_ID env var")
		os.Exit(1)
	}

	if wrappenToken {
		wrappenTokenFile := os.Getenv("APPROLE_WRAPPEN_TOKEN_FILE")
		err := checkTokenFileExists(wrappenTokenFile)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		type wrappedTokenInfo struct{
			Token string `json:"token"`
		}

		data := wrappedTokenInfo{}
		file, _ := ioutil.ReadFile(wrappenTokenFile)
		_ = json.Unmarshal([]byte(file), &data)
		wrappenSecretID := &auth.SecretID{FromString: data.Token}
		withWrappinToken, err := auth.NewAppRoleAuth(roleID, wrappenSecretID, auth.WithWrappingToken())
		if err != nil {
			fmt.Println("unable to initialize AppRole auth method: %w", err)
		}
		return &vaultStorage{
			client:     client,
			appRole:    withWrappinToken,
			secretPath: "secret/data/sample/go_webapp",
			secretKey:  "password",
		}
	}

	unwrappenTokenFile := os.Getenv("APPROLE_UNWRAPPEN_TOKEN_FILE")
	err := checkTokenFileExists(unwrappenTokenFile)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	unwrappenSecretID := &auth.SecretID{FromFile: unwrappenTokenFile}
	withUnwrappinToken, err := auth.NewAppRoleAuth(roleID, unwrappenSecretID)
	if err != nil {
		fmt.Println("unable to initialize AppRole auth method: %w", err)
	}
	return &vaultStorage{
		client:     client,
		appRole:    withUnwrappinToken,
		secretPath: "secret/data/sample/go_webapp",
		secretKey:  "username",
	}
}

func getSecretWithAppRole(storage *vaultStorage) (string, error) {

	authInfo, err := storage.client.Auth().Login(context.TODO(), storage.appRole)
	if err != nil {
		return "", fmt.Errorf("unable to login to AppRole auth method: %w", err)
	}
	if authInfo == nil {
		return "", fmt.Errorf("no auth info was returned after login")
	}

	// get secret
	secret, err := storage.client.Logical().Read(storage.secretPath)
	if err != nil {
		return "", fmt.Errorf("unable to read secret: %w", err)
	}

	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("data type assertion failed: %T %#v", secret.Data["data"], secret.Data["data"])
	}

	value, ok := data[storage.secretKey].(string)
	if !ok {
		return "", fmt.Errorf("value type assertion failed: %T %#v", data[storage.secretKey], data[storage.secretKey])
	}

	return value, nil
}

func checkTokenFileExists(fileName string) error {
	if len(fileName) > 0 {
		_, err := os.Stat(fileName)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("no token file was provided in APPROLE_TOKEN_FILE env var")
	}
	return nil
}


func main() {

	client, err := newClient()
	if err != nil {
		fmt.Println(err.Error())
	}
	unwrappinStorage := newStorage(client, false)
	wrappinStorage := newStorage(client, true)
	password, err := getSecretWithAppRole(wrappinStorage)
	if err != nil {
		fmt.Println(err.Error())
	}

	user, err := getSecretWithAppRole(unwrappinStorage)
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println(password, user)
}
