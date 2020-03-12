package cmd

/*
 Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

import (
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strconv"

	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	v4http "github.com/crunchydata/postgres-operator/pgo-client/v4/http"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "pgo",
	Short: "The pgo command line interface.",
	Long:  `The pgo command line interface lets you create and manage PostgreSQL clusters.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	fmt.Println("Execute called")

	if err := RootCmd.Execute(); err != nil {
		log.Debug(err.Error())
		os.Exit(-1)
	}

}

func init() {

	cobra.OnInitialize(initConfig)
	log.Debug("init called")
	GREEN = color.New(color.FgGreen).SprintFunc()
	YELLOW = color.New(color.FgYellow).SprintFunc()
	RED = color.New(color.FgRed).SprintFunc()

	// Go currently guarantees an error when attempting to load OS_TRUST for
	// windows-based systems (see https://golang.org/issue/16736)
	defExclOSTrust := (runtime.GOOS == "windows")

	RootCmd.PersistentFlags().StringVarP(&Namespace, "namespace", "n", "", "The namespace to use for pgo requests.")
	RootCmd.PersistentFlags().StringVar(&APIServerURL, "apiserver-url", "", "The URL for the PostgreSQL Operator apiserver that will process the request from the pgo client.")
	RootCmd.PersistentFlags().StringVar(&PGO_CA_CERT, "pgo-ca-cert", "", "The CA Certificate file path for authenticating to the PostgreSQL Operator apiserver.")
	RootCmd.PersistentFlags().StringVar(&PGO_CLIENT_KEY, "pgo-client-key", "", "The Client Key file path for authenticating to the PostgreSQL Operator apiserver.")
	RootCmd.PersistentFlags().StringVar(&PGO_CLIENT_CERT, "pgo-client-cert", "", "The Client Certificate file path for authenticating to the PostgreSQL Operator apiserver.")
	RootCmd.PersistentFlags().BoolVar(&PGO_DISABLE_TLS, "disable-tls", false, "Disable TLS authentication to the Postgres Operator.")
	RootCmd.PersistentFlags().BoolVar(&EXCLUDE_OS_TRUST, "exclude-os-trust", defExclOSTrust, "Exclude CA certs from OS default trust store")
	RootCmd.PersistentFlags().BoolVar(&DebugFlag, "debug", false, "Enable additional output for debugging.")

}

func initConfig() {
	if DebugFlag {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug flag is set to true")
	}

	if APIServerURL == "" {
		APIServerURL = os.Getenv("PGO_APISERVER_URL")
		if APIServerURL == "" {
			fmt.Println("Error: The PGO_APISERVER_URL environment variable or the --apiserver-url flag needs to be supplied.")
			os.Exit(-1)
		}
	}
	log.Debugf("in initConfig with url=%s", APIServerURL)

	baseURL, err := url.Parse(APIServerURL)
	if err != nil {
		log.Fatalf("could not parse api server url: %s", err.Error())
	}

	tmp := os.Getenv("PGO_NAMESPACE")
	if tmp != "" {
		PGONamespace = tmp
		log.Debugf("using PGO_NAMESPACE env var %s", tmp)
	}

	// Get the pgouser and password information
	SetSessionUserCredentials()

	// Setup the API HTTP client based on TLS enablement
	if noTLS, _ := strconv.ParseBool(os.Getenv("DISABLE_TLS")); noTLS || PGO_DISABLE_TLS {
		log.Debug("setting up httpclient without TLS")
		if err != nil {
			log.Fatal("failed to parse api server url: %s", err.Error())
		}
		httpclient = NewAPIClient()
	} else {
		log.Debug("setting up httpclient with TLS")
		if hc, err := NewAPIClientTLS(); err != nil {
			log.Fatalf("failed to set up TLS client: %s", err)
		} else {
			httpclient = hc
		}
	}

	clientConfig := v4http.Config{
		Address: baseURL,
		Credentials: v4http.Credentials{
			Username: SessionCredentials.Username,
			Password: SessionCredentials.Password,
		},
	}

	if noTLS, _ := strconv.ParseBool(os.Getenv("DISABLE_TLS")); !noTLS || !PGO_DISABLE_TLS {
		transport, err := GetTLSTransport()
		if err != nil {
			log.Fatalf("failed to setup TLS client: %s", err.Error())
		}
		clientConfig.Transport = transport
	}

	apiClient, err = v4http.NewAPI(clientConfig)

	if err != nil {
		log.Fatalf("could not initialize API client: %s", err.Error())
	}

	if os.Getenv("GENERATE_BASH_COMPLETION") != "" {
		generateBashCompletion()
	}
}

func generateBashCompletion() {
	log.Debugf("generating bash completion script")
	file, err2 := os.Create("/tmp/pgo-bash-completion.out")
	if err2 != nil {
		fmt.Println("Error: ", err2.Error())
	}
	defer file.Close()
	RootCmd.GenBashCompletion(file)
}
