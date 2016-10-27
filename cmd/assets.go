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
	"strings"

	"github.com/Sirupsen/logrus"
	bolt "github.com/johnnadratowski/golang-neo4j-bolt-driver"
	"github.com/kkirsche/trace2neo/cypherBuilder"
	"github.com/kkirsche/trace2neo/trace2neolib"
	"github.com/spf13/cobra"
)

var (
	successfulResolutions,
	failedResolutions []string
	write                    bool
	err                      error
	conn                     bolt.Conn
	username, password, host string
	port                     int
)

// assetsCmd represents the assets command
var assetsCmd = &cobra.Command{
	Use:   "assets",
	Short: "Resolves CIDR blocks to Neo4j Nodes if in DNS",
	Long: `Resolves a CIDR block or a list of CIDR blocks from DNS, and then builds
a cypher query for use with Neo4j. The tool will output the assets.cypher file to
the current working directory (./assets.cypher).

trace2neo assets <cidr>

trace2neo assets <cidr>,<cidr>,<cidr>

trace2neo assets <cidr>, <cidr>, <cidr>
`,
	Run: func(cmd *cobra.Command, args []string) {
		// args should be an array of CIDR notation addresses
		if !write {
			driver := bolt.NewDriver()
			conn, err = driver.OpenNeo(fmt.Sprintf("bolt://%s:%s@%s:%d", username, password, host, port))
			if err != nil {
				logrus.WithError(err).Errorln("Failed to open connection to Neo4j. Is it running?")
				return
			}
			defer conn.Close()
		}
		for _, cidr := range args {
			ip, ipnet, err := net.ParseCIDR(strings.TrimSpace(cidr))
			if err != nil {
				logrus.WithError(err).Errorf("Failed to parse %s as CIDR block. Skipping...", cidr)
				continue
			}

			var ips []string
			for ipaddr := ip.Mask(ipnet.Mask); ipnet.Contains(ipaddr); inc(ipaddr) {
				ips = append(ips, ipaddr.String())
			}
			availableIPs := ips[1 : len(ips)-1]

			t, err := cypherBuilder.GetAssetTemplate()
			if err != nil {
				logrus.WithError(err).Errorln("Failed to get asset template. Exiting")
				return
			}

			lenIPs := len(availableIPs)
			for i, availableIP := range availableIPs {
				if i%250 == 0 {
					logrus.Infof("Currently resolving %s (#%d of #%d)...", availableIP, i, lenIPs)
				}
				resolved, loopErr := trace2neolib.ResolveAddr(availableIP)
				if loopErr != nil {
					failedResolutions = append(failedResolutions, availableIP)
				}

				assets := trace2neolib.ResolvedAddrToAsset(resolved, availableIP, i)
				if len(assets) > 0 {
					for _, asset := range assets {
						if !write {
							stmt, innerLoopErr := conn.PrepareNeo("CREATE (n:Unknown {name: {name}, ip: {ip}})")
							if innerLoopErr != nil {
								logrus.WithError(err).Errorln("Failed to create statement.")
								return
							}
							_, innerLoopErr = stmt.ExecNeo(map[string]interface{}{"shortName": asset.ShortName, "label": asset.Label, "name": asset.Name, "ip": asset.IPAddr})
							if innerLoopErr != nil {
								logrus.WithError(err).Errorln("Failed to execute create statement.")
								return
							}
							stmt.Close()
						}

						if asset != nil {
							assetString, innerLoopErr := cypherBuilder.BuildAsset(t, asset)
							if innerLoopErr != nil {
								logrus.WithError(innerLoopErr).Errorln("Failed to build asset %s", asset.IPAddr)
								continue
							}
							successfulResolutions = append(successfulResolutions, assetString)
						}
					}
				}
			}
		}

		if write {
			wd, err := os.Getwd()
			if err != nil {
				logrus.WithError(err).Errorln("Failed to get current working directory")
			}
			fp := wd + "/assets.cypher"
			logrus.Infof("Writing cypher query to %s", fp)
			cypherBuilder.WriteAssetsToFile(successfulResolutions, fp)
		}

		if verbose {
			logrus.Debugln("Successfully resolved:")
			for _, success := range successfulResolutions {
				logrus.Debugln(success)
			}
		}

		if verbose {
			logrus.Debugln("Failed to resolve:")
			for _, failed := range failedResolutions {
				logrus.Debugln(failed)
			}
		}
	},
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func init() {
	RootCmd.AddCommand(assetsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	assetsCmd.PersistentFlags().StringVarP(&username, "username", "u", "neo4j", "Neo4j username")
	assetsCmd.PersistentFlags().StringVarP(&password, "password", "p", "Example", "Neo4j password")
	assetsCmd.PersistentFlags().StringVarP(&host, "bolt-host", "b", "localhost", "Neo4j host")
	assetsCmd.PersistentFlags().IntVarP(&port, "port", "o", 7687, "Neo4j port")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	assetsCmd.Flags().BoolVarP(&write, "write", "w", false, "Write to file rather than to Neo4j directly")

}
