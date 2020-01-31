// Copyright (c) Remi Ferrand
//
// Author(s): Remi Ferrand <remi.ferrand_at_cc.in2p3.fr>, 2020
//
// This software is governed by the CeCILL-B license under French law and
// abiding by the rules of distribution of free software.  You can  use,
// modify and/ or redistribute the software under the terms of the CeCILL-B
// license as circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and  rights to copy,
// modify and redistribute granted by the license, users are provided only
// with a limited warranty  and the software's author,  the holder of the
// economic rights,  and the successive licensors  have only  limited
// liability.
//
// In this respect, the user's attention is drawn to the risks associated
// with loading,  using,  modifying and/or developing or reproducing the
// software by the user in light of its specific status of free software,
// that may mean  that it is complicated to manipulate,  and  that  also
// therefore means  that it is reserved for developers  and  experienced
// professionals having in-depth computer knowledge. Users are therefore
// encouraged to load and test the software's suitability as regards their
// requirements in conditions enabling the security of their systems and/or
// data to be ensured and,  more generally, to use and operate it in the
// same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL-B license and that you accept its terms.

package cmd

import (
	"fmt"
	"io/ioutil"
	"log/syslog"
	"os"
	"path/filepath"
	"sync"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/riton/puppet-environment-cache-invalidate/puppetapi"
	log "github.com/sirupsen/logrus"
	logrus_syslog "github.com/sirupsen/logrus/hooks/syslog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "puppet-environment-cache-invalidate",
	Short: "Invalidate Puppetserver environment cache",
	Long: `CLI that invalidates puppetserver environment cache simultaneously on multiple puppet servers.

	See https://puppet.com/docs/puppetserver/latest/admin-api/v1/environment-cache.html for more informations

	Example:
	To invalidate the environment cache for the 'production' environment
	$ /usr/bin/puppet-environment-cache-invalidate production
	`,

	SilenceUsage: true,
	Args: func(cmd *cobra.Command, args []string) error {
		// TODO: Make it optional
		if len(args) != 1 {
			return fmt.Errorf("requires a Puppet environment")
		}
		return nil
	},
	RunE: runE,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initLogging()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.puppet-environment-cache-invalidate.yaml)")
	rootCmd.PersistentFlags().Bool("debug", false, "enable debug")
	rootCmd.PersistentFlags().Bool("log-syslog", false, "log to syslog")
	rootCmd.PersistentFlags().Bool("log-json", false, "log in JSON format")

	viper.BindPFlag("log-syslog", rootCmd.PersistentFlags().Lookup("log-syslog"))
	viper.BindPFlag("log-json", rootCmd.PersistentFlags().Lookup("log-json"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
}

func initLogging() {
	useSyslog := viper.GetBool("log-syslog")
	jsonLogFmt := viper.GetBool("log-json")
	enableDebug := viper.GetBool("debug")

	if enableDebug {
		log.SetLevel(log.DebugLevel)
	}

	if jsonLogFmt {
		log.SetFormatter(&log.JSONFormatter{})
	}

	if useSyslog {
		logLevel := syslog.LOG_INFO
		if enableDebug {
			logLevel = syslog.LOG_DEBUG
		}

		hook, err := logrus_syslog.NewSyslogHook("", "", logLevel, "puppet-environment-cache-invalidate")
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: setting up syslog logging: %s\n", err)
			os.Exit(1)
		}

		log.AddHook(hook)
		log.SetOutput(ioutil.Discard)
	} else {
		log.SetOutput(os.Stdout)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

	//viper.SetConfigType("yaml")

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("/etc")

		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".puppet-environment-cache-invalidate" (without extension).
		viper.AddConfigPath(filepath.Join(home, ".puppet-environment-cache-invalidate"))
		viper.SetConfigName("puppet-environment-cache-invalidate")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

type jobResult struct {
	PuppetServer      string
	PuppetEnvironment string
	Error             error
}

func runE(cmd *cobra.Command, args []string) error {
	puppetEnvironment := args[0]
	puppetServers := viper.GetStringSlice("puppetservers")

	// auth settings
	x509ClientCertFile := viper.GetString("auth.certfile")
	x509PrivateKeyFile := viper.GetString("auth.pkfile")
	x509CABundleFile := viper.GetString("auth.ca-bundle")

	if len(puppetServers) == 0 {
		log.Fatal("empty puppetservers list")
	}

	httpClient, err := puppetapi.NewTLSAuthenticatedHTTPClient(x509ClientCertFile,
		x509CABundleFile, x509PrivateKeyFile)
	if err != nil {
		log.Fatalf("building authenticated HTTP client: %s", err)
	}

	jobDoneChan := make(chan jobResult, len(puppetServers))
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(puppetServers))

	for _, puppetServer := range puppetServers {
		go func(server string) {

			api := puppetapi.NewPuppetAPIWithHTTPClient(server, httpClient)
			err = api.InvalidateEnvironmentCache(puppetEnvironment)
			jobDoneChan <- jobResult{
				PuppetServer:      server,
				PuppetEnvironment: puppetEnvironment,
				Error:             err,
			}

			waitGroup.Done()
		}(puppetServer)
	}

	waitGroup.Wait()
	close(jobDoneChan)

	var hasError bool
	for result := range jobDoneChan {
		if result.Error != nil {
			hasError = true
			log.WithFields(log.Fields{
				"server":      result.PuppetServer,
				"environment": result.PuppetEnvironment,
				"error":       result.Error,
			}).Error("fail to invalidate environment cache")
		} else {
			log.WithFields(log.Fields{
				"server":      result.PuppetServer,
				"environment": result.PuppetEnvironment,
			}).Info("environment cache invalidated")
		}
	}

	if hasError {
		return fmt.Errorf("failed to invalidate some servers cache")
	}

	return nil
}
