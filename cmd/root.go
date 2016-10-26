// Copyright © 2016 Kevin Kirsche <kev.kirsche[at]gmail.com>
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
	"net"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/kkirsche/trace2neo/traceroute"
	"github.com/spf13/cobra"
)

var verbose bool

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "trace2neo",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		}
		for _, ip := range args {
			netip := net.ParseIP(ip)
			if netip == nil {
				logrus.Errorf("Argument %s was not an IP. Skipping...", ip)
				continue
			}
			result, err := traceroute.RunTraceroute(netip)
			if err != nil {
				logrus.WithError(err).Errorln("Failed to run traceroute.")
				continue
			}

			processedResults, err := traceroute.ProcessTracerouteOutput(result)
			if err != nil {
				logrus.WithError(err).Errorln("Failed to process traceroute output.")
			}
			for _, processedResult := range processedResults {
				logrus.Debugf("Hop: %s, Destination: %s, DNS Host: %s, IP: %s, RTT1: %s, RTT2: %s, RTT3: %s",
					processedResult.Hop, processedResult.Destination, processedResult.DNSName, processedResult.IP,
					processedResult.RTT1, processedResult.RTT2, processedResult.RTT3)
			}
		}
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
	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags, which, if defined here,
	// will be global for your application.

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose mode")
}
