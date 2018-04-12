// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os"

	"github.com/joycn/puckgo/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var cfg *config.Config = &config.Config{}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "puckgo",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		start(cfg)
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags, which, if defined here,
	// will be global for your application.

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.dnsforward.yaml)")
	RootCmd.PersistentFlags().BoolVarP(&config.PublicService, "public", "p", false, "whether run in public service mode")
	//RootCmd.PersistentFlags().StringVar(&cfg.LogLevel, "loglevel", "info", "set log level")
	//RootCmd.PersistentFlags().StringVar(&cfg.DataSource, "data", config.DefaultDataSource, "data source used for url query")
	//RootCmd.PersistentFlags().BoolVar(&cfg.DNS.DefaultAction, "defaultdrop", false, "data source used for url query")
	//RootCmd.PersistentFlags().StringVar(&cfg.DNS.DefaultServer, "default", config.DefaultServer, "default dns server")
	//RootCmd.PersistentFlags().StringVar(&cfg.DNS.Listen, "listen", config.DefaultListen, "default dns server")
	//RootCmd.PersistentFlags().StringVar(&cfg.DNS.ExceptiveServer, "except", config.ExceptiveServer, "exceptive dns server")
	//RootCmd.PersistentFlags().StringVar(&cfg.TransparentProxy.ProxyListen, "proxylisten", config.DefaultProxyListen, "listen address for proxy")
	//RootCmd.PersistentFlags().IntVar(&cfg.TransparentProxy.ProxyTimeout, "timeout", config.DefaultProxyTimeout, "default timeout for proxy connection")
	//RootCmd.PersistentFlags().StringVar(&cfg.TransparentProxy.ProxyUpstream, "upstream", "", "default timeout for proxy connection")
	//RootCmd.MarkPersistentFlagRequired("upstream")
	//RootCmd.PersistentFlags().Lookup("defaultdrop").NoOptDefVal = true
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".puckgo") // name of config file (without extension)
	viper.AddConfigPath("$HOME")   // adding home directory as first search path
	viper.AutomaticEnv()           // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
		if err := viper.Unmarshal(cfg); err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(-1)
		} else {
			fmt.Println(cfg)
		}
	} else {
		fmt.Printf("%v\n", err)
		os.Exit(-1)

	}

	//if cfg.DNS.DefaultServer == "" {
	//fmt.Println("upstream not set")
	//os.Exit(-1)
	//}

	if cfg.TransparentProxy.ProxyUpstream == "" {
		fmt.Println("upstream not set")
		os.Exit(-1)
	}
}
