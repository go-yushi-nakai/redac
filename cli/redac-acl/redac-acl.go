package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Songmu/prompter"
	"github.com/go-yushi-nakai/redac"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.PersistentFlags().BoolP("version", "v", false, "show version info")
	rootCmd.AddCommand(showUsersIDCmd)
}

var rootCmd = &cobra.Command{
	Use:   "redac-acl",
	Short: "access control list for redash as command",
	Run: func(cmd *cobra.Command, args []string) {
		if v, _ := cmd.Flags().GetBool("version"); v {
			fmt.Fprintf(os.Stderr, "%s\n", redac.GetVersion())
			os.Exit(-1)
		}
	},
}

var showUsersIDCmd = &cobra.Command{
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

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
