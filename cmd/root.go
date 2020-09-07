/*
Copyright © 2020 Ulrich Wisser

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

import (
	"log"
	"net"

	"github.com/miekg/dns"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string
var strict bool
var resolvers []string
var concurrent int
var verbose int

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "zone6",
	Short: "IPv6 statistics for your zone",
	Long: `
This application takes a zone file and computes
several IPv6 statistics from it.

Attention! Depending on the size of your zone computation can
take some time. The application tries to be nice to your resolver.`,

	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().CountVarP(&verbose, "verbose", "v", "print information while processing (can be given several times)")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.zone6)")
	rootCmd.PersistentFlags().BoolVar(&strict, "strict", false, "Perform check on full IPv6 reachability (no implemented)")
	rootCmd.PersistentFlags().StringSliceVarP(&resolvers, "resolver", "r", []string{}, "IPv4 or IPv6 number of an resolver (can be given multiple times)")
	rootCmd.PersistentFlags().IntVar(&concurrent, "concurrent", 10, "Number of concurrent resolvings (default: 10)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			log.Fatal(err)
		}

		// Search config in home directory with name ".zone6" (without extension).
		viper.SetConfigName(".zone6")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(home)
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using config file:", viper.ConfigFileUsed())
	}

	// if no resolvers are given, try to get system resolvers
	if len(resolvers) == 0 {
		resolvers = getSystemResolvers()
	}

	// now format all resolvers to be used later
	// this will add port number if needed
	resolvers = formatResolvers(resolvers)

	// no can do without resolvers
	if len(resolvers) == 0 {
		log.Fatal("No resolvers found")
	}
}

// getResolvers will read the list of resolvers from /etc/resolv.conf
func getSystemResolvers() []string {
	resolvers := make([]string, 0)

	conf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if conf == nil {
		log.Fatalf("Cannot read /etc/resolv.conf: %s\n", err)
	}
	for _, server := range conf.Servers {
		resolvers = append(resolvers, server)
	}
	if len(resolvers) == 0 {
		log.Fatal("No resolvers found.")
	}
	return resolvers
}

// resolvers need to be of format
// IPv4: 1.1.1.1:53
// IPv6: [1::1]:53
//ipv4withPort := regexp.MustCompile("^(?:(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])(\\.(?!$)|$)){4}:\\d$")
//ipv4 := regexp.MustCompile("^(?:(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])(\\.(?!$)|$)){4}$")

func formatResolvers(resolvers []string) []string {
	newresolvers := make([]string, 0)
	for _, resolver := range resolvers {
		newresolver := formatResolver(resolver)
		if len(newresolver) > 0 {
			newresolvers = append(newresolvers, newresolver)
		}
	}
	return newresolvers
}

func formatResolver(resolver string) string {
	addr, err := net.ResolveUDPAddr("udp", resolver)
	if err == nil {
		if addr.Port == 0 {
			addr.Port = 53 // DNS standard port
		}
		return addr.String()
	}
	ip := net.ParseIP(resolver)
	if ip != nil {
		return net.JoinHostPort(ip.String(), "53") // DNS standard port
	}
	if verbose > 0 {
		log.Printf("Not a valid resolver: %s", resolver)
	}
	return ""
}
