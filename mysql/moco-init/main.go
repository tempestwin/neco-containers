package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

const (
	defaultBaseDir = "/usr/local/mysql"
	defaultDataDir = "/var/mysql"
	defaultConfDir = "/etc/mysql-conf.d"

	initializedFile = "moco-initialized"
)

var config struct {
	baseDir string
	dataDir string
	confDir string

	podName  string
	baseID   uint32
	podIndex uint32
}

//go:embed my.cnf
var mycnfTmpl string

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "moco-init SERVER_ID_BASE",
	Short: "initialize MySQL",
	Long: `moco-init initializes MySQL data directory and create a
configuration snippet to give instance specific configuration values
such as server_id and admin_address.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return subMain(args[0])
	},
}

func subMain(serverIDBase string) error {

	config.podName = os.Getenv("POD_NAME")
	if len(config.podName) == 0 {
		return fmt.Errorf("no POD_NAME environment variable")
	}

	fields := strings.Split(config.podName, "-")
	if len(fields) < 2 {
		return fmt.Errorf("bad POD_NAME: %s", config.podName)
	}

	indexUint64, err := strconv.ParseUint(fields[len(fields)-1], 10, 32)
	if err != nil {
		return fmt.Errorf("bad POD_NAME %s", config.podName)
	}
	config.podIndex = uint32(indexUint64)

	baseUint64, err := strconv.ParseUint(serverIDBase, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid server base ID: %s: %w", os.Args[1], err)
	}
	config.baseID = uint32(baseUint64)

	_, err = os.Stat(filepath.Join(config.dataDir, initializedFile))
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	return createConf()
}

func createConf() error {
	tmpl := template.Must(template.New("my.cnf").Parse(mycnfTmpl))

	v := struct {
		ServerID     uint32
		AdminAddress string
	}{
		ServerID:     config.baseID + config.podIndex,
		AdminAddress: config.podName,
	}

	f, err := os.OpenFile(filepath.Join(config.confDir, "my.cnf"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create my.cnf file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, v); err != nil {
		return err
	}
	return f.Sync()
}

func init() {
	rootCmd.Flags().StringVar(&config.baseDir, "base-dir", defaultBaseDir, "The base directory for MySQL.")
	rootCmd.Flags().StringVar(&config.dataDir, "data-dir", defaultDataDir, "The data directory for MySQL.  Data will be stored in a subdirectory named 'data'")
	rootCmd.Flags().StringVar(&config.confDir, "conf-dir", defaultConfDir, "The directory where configuration file is created.")
}
