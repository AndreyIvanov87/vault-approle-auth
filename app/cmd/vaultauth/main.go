package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"
	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/approle"
	"golang.org/x/net/http2"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type vaultStorage struct{
	client *vault.Client
	appRole *auth.AppRoleAuth
	secretPath string
	secretKey string
}

func newConfig() *vault.Config{
	vaultAddr := os.Getenv("APPROLE_VAULT_ADDR")
	if vaultAddr == "" {
		fmt.Println("no role ID was provided in APPROLE_VAULT_ADDR env var")
		os.Exit(1)
	}

	fmt.Println("init connection to vault in addr %s", vaultAddr)

	config := &vault.Config{
		Address:      vaultAddr,
		HttpClient:   cleanhttp.DefaultPooledClient(),
		Timeout:      time.Second * 60,
		MinRetryWait: time.Millisecond * 1000,
		MaxRetryWait: time.Millisecond * 1500,
		MaxRetries:   2,
		Backoff:      retryablehttp.LinearJitterBackoff,
	}

	transport := config.HttpClient.Transport.(*http.Transport)
	transport.TLSHandshakeTimeout = 10 * time.Second
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if err := http2.ConfigureTransport(transport); err != nil {
		config.Error = err
		return config
	}

	if err := config.ReadEnvironment(); err != nil {
		config.Error = err
		return config
	}

	config.HttpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	return config
}

func newClient() (*vault.Client, error){
	config := newConfig()
	tlsConfig := vault.TLSConfig{
		CACert:        "",
		CAPath:        "",
		ClientCert:    "",
		ClientKey:     "",
		TLSServerName: "",
		Insecure:      true,
	}
	if tls := len(os.Getenv("APPROLE_VAULT_TLS")); tls > 0 {
		fmt.Println("vault tls connection enabling")
		config.ConfigureTLS(&tlsConfig)
	}

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize Vault client: %w", err)
	}
	return client, nil
}


func newStorage(client *vault.Client, wrappenToken bool) *vaultStorage {
	roleID := os.Getenv("APPROLE_ROLE_ID")
	fmt.Println("app role id is: %s", roleID)
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
		fmt.Println("reading from file: %s", file)
		_ = json.Unmarshal([]byte(file), &data)
		fmt.Println("reading token is: %s", data.Token)
		wrappenSecretID := &auth.SecretID{FromString: data.Token}
		withWrappinToken, err := auth.NewAppRoleAuth(roleID, wrappenSecretID, auth.WithWrappingToken())
		if err != nil {
			fmt.Println("unable to initialize AppRole auth method: %w", err)
		}
		return &vaultStorage{
			client:     client,
			appRole:    withWrappinToken,
			secretPath: "secret/data/k11s/demo/app/service",
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
		secretPath: "secret/data/k11s/demo/app/service",
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
		return fmt.Errorf("no token file was provided in env var")
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
