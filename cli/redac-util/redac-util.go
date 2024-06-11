package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/Songmu/prompter"
	"github.com/go-yushi-nakai/redac"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)

	configCmd.AddCommand(listCmd)
	configCmd.AddCommand(addCmd)
	configCmd.AddCommand(delCmd)
	rootCmd.AddCommand(showUsersCmd)
}

var rootCmd = &cobra.Command{
	Use:   "redac-config",
	Short: "utilitiy command for redac",
}

var versionCmd = &cobra.Command{
	Use: "version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(redac.GetVersion())
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "configuration for redac",
}

var listCmd = &cobra.Command{
	Use: "list",
	Run: func(cmd *cobra.Command, args []string) {
		redac.LoadConfig()
		c, err := redac.LoadConfig()
		if err != nil {
			fmt.Printf("failed to load config: %s\n", err)
			return
		}
		for _, configCtx := range c.Contexts {
			fmt.Printf("%s: endpoint=%s, data_source_id=%d\n", configCtx.Name, configCtx.Endpoint, configCtx.DataSourceID)
		}
	},
}

var addCmd = &cobra.Command{
	Use: "add",
	Run: func(cmd *cobra.Command, args []string) {
		logger, err := redac.NewLogger("info")
		if err != nil {
			fmt.Printf("failed to create logger: %s\n", err)
			return
		}

		name := prompter.Prompt("context name", "")
		endpoint := prompter.Prompt("redash URL", "")
		apiKey := prompter.Password("API Key")

		rc, err := redac.NewRedashClient(endpoint, apiKey, logger)
		if err != nil {
			fmt.Printf("failed to create redash client: %s\n", err)
			return
		}
		sources, err := rc.GetDataSources(context.Background())
		if err != nil {
			fmt.Printf("failed to get data sources: %s\n", err)
			return
		}
		fmt.Printf("list of data sources from %s:\n", endpoint)
		for _, source := range sources {
			id, ok := source["id"].(float64)
			if !ok {
				fmt.Println("failed to parse source ID")
				return
			}
			name, ok := source["name"].(string)
			if !ok {
				fmt.Println("failed to parse source name")
				return
			}
			fmt.Printf("  id=%d: %s\n", int(id), name)
		}
		dsIDStr := prompter.Prompt("select source ID", "")
		dsID, err := strconv.Atoi(dsIDStr)
		if err != nil {
			fmt.Printf("failed to parse source ID: %s\n", err)
			return
		}
		if err := redac.AddConfigContext(name, endpoint, apiKey, dsID); err != nil {
			fmt.Printf("failed to add context: %s\n", err)
			return
		}
	},
}

var delCmd = &cobra.Command{
	Use: "del",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("no context name specified")
			return
		}

		for _, target := range args {
			if err := redac.DeleteConfigContext(target); err != nil {
				fmt.Printf("failed to delete context: %s\n", err)
			}
			fmt.Printf("context name=%s deleted\n", target)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}

var showUsersCmd = &cobra.Command{
	Use: "show-users",
	Run: func(cmd *cobra.Command, args []string) {
		logger, err := redac.NewLogger("debug") // FIXME: debug -> info
		if err != nil {
			fmt.Printf("failed to create logger: %s\n", err)
			return
		}

		c, err := redac.LoadConfig()
		if err != nil {
			fmt.Printf("failed to load config: %s\n", err)
			return
		}

		contextName := args[0]
		cc := c.Contexts[contextName]

		rc, err := redac.NewRedashClient(cc.Endpoint, cc.APIKey, logger)
		if err != nil {
			fmt.Printf("failed to create redash client: %s\n", err)
			return
		}

		searchWord := prompter.Prompt("search word", "")

		users, err := rc.GetUsers(context.Background(), searchWord)
		if err != nil {
			fmt.Printf("failed to get users: %s\n", err)
			return
		}

		jsonData, err := json.MarshalIndent(users, "", "    ")
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(string(jsonData))

	},
}
