package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/go-yushi-nakai/redac"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.SetUsageTemplate(usageForDefault)
	rootCmd.PersistentFlags().BoolP("version", "v", false, "show version info")
	rootCmd.PersistentFlags().StringP("eval", "e", "", "evaluate sql")
	rootCmd.PersistentFlags().Bool("no-limit", false, "disalbe auto-limit flag in redash")
	rootCmd.PersistentFlags().Bool("no-header", false, "hide header line from output")
	rootCmd.PersistentFlags().StringP("format", "f", "table1", "output format table1/table2/csv/json/yaml (default:table1")
	rootCmd.PersistentFlags().StringP("timeout", "t", "10s", "timeout")
	rootCmd.PersistentFlags().StringP("loglevel", "l", "warn", "loglevel(debug/info/warn/error)")
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		NewRedacCommand(cmd, args)
		cmd.Usage()
	})

}

const (
	usageForDefault = `Usage:{{if .Runnable}}
  {{.Use}} [flags...] -e <query_string> <context_name> [args...]
  {{.Use}} [flags...] <query_file> <context_name> [args...]
{{end}}
Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
`
)

func usageString(baseArgs, additionalOpts string) string {
	baseArgs = strings.ReplaceAll(baseArgs, "{{", `{{ "{{`)
	baseArgs = strings.ReplaceAll(baseArgs, "}}", `}}" }}`)
	opts := fmt.Sprintf("[flags...] %s <context_name>", baseArgs)
	if additionalOpts != "" {
		opts += " " + additionalOpts
	}
	s := `Usage:{{if .Runnable}}
  {{.Use}} %s
{{end}}
Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
`
	return fmt.Sprintf(s, opts)
}

var rootCmd = &cobra.Command{
	Use:   "redac",
	Short: "tool for redash as command",
	Run: func(cmd *cobra.Command, args []string) {
		if v, _ := cmd.Flags().GetBool("version"); v {
			fmt.Fprintf(os.Stderr, "%s\n", redac.GetVersion())
			os.Exit(-1)
		}
		rc, err := NewRedacCommand(cmd, args)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			fmt.Fprintln(os.Stderr, "")
			cmd.Usage()
			os.Exit(1)
		}
		defer rc.cancel()

		if err, withUsage := rc.Run(cmd, args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			if withUsage {
				fmt.Fprintln(os.Stderr, "")
				cmd.Usage()
			}
			os.Exit(2)
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}

type RedacCommand struct {
	ctx         context.Context
	cancel      context.CancelFunc
	logger      *slog.Logger
	config      *redac.ConfigFile
	query       *redac.Query
	noLimit     bool
	noHeader    bool
	renderer    redac.Renderer
	contextName string
	queryArgs   []string
}

func NewRedacCommand(cmd *cobra.Command, args []string) (*RedacCommand, error) {
	c := &RedacCommand{}

	timeoutStr, err := cmd.Flags().GetString("timeout")
	if err != nil {
		return nil, fmt.Errorf("failed to get timeout option: %w", err)
	}
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timeout: %w", err)
	}
	c.ctx, c.cancel = context.WithTimeout(context.Background(), timeout)

	levelStr, err := cmd.Flags().GetString("loglevel")
	if err != nil {
		return nil, fmt.Errorf("failed to get loglevel option: %w", err)
	}
	logger, err := redac.NewLogger(levelStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}
	noLimit, err := cmd.Flags().GetBool("no-limit")
	if err != nil {
		return nil, fmt.Errorf("failed to get no-limit option: %w", err)
	}
	c.noLimit = noLimit

	noHeader, err := cmd.Flags().GetBool("no-header")
	if err != nil {
		return nil, fmt.Errorf("failed to get no-limit option: %w", err)
	}
	c.noHeader = noHeader

	c.logger = logger

	conf, err := redac.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	c.config = conf
	restArgs := args
	if cmd.Flags().Changed("eval") {
		evalStr, err := cmd.Flags().GetString("eval")
		if err != nil {
			return nil, fmt.Errorf("failed to get eval str: %w", err)
		}
		q, err := redac.NewQuery(evalStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse query: %w", err)
		}
		c.query = q

		usageArgs := c.query.GetParameterStringForUsage()
		cmd.SetUsageTemplate(usageString(fmt.Sprintf(`-e "%s"`, evalStr), usageArgs))

		if len(restArgs) == 0 {
			return nil, fmt.Errorf("no context name specified")
		}
	}

	if c.query == nil {
		if len(restArgs) == 0 {
			return nil, fmt.Errorf("no query file specified")
		}
		filePath := restArgs[0]
		q, err := redac.LoadQueryFromFile(filePath)
		restArgs = restArgs[1:]
		if err != nil {
			return nil, fmt.Errorf("failed to get query from file: %w", err)
		}
		c.query = q
		usageArgs := c.query.GetParameterStringForUsage()
		cmd.SetUsageTemplate(usageString(filePath, usageArgs))

		if len(args) == 1 {
			return nil, fmt.Errorf("no context name specified")
		}
	}
	c.contextName = restArgs[0]
	restArgs = restArgs[1:]
	c.queryArgs = restArgs

	formatStr, err := cmd.Flags().GetString("format")
	if err != nil {
		return nil, fmt.Errorf("failed to get format str: %w", err)
	}
	switch formatStr {
	case "table1":
		c.renderer = &redac.TableRenderer{TableType: redac.TableType1}
	case "table2":
		c.renderer = &redac.TableRenderer{TableType: redac.TableType2}
	case "csv":
		c.renderer = &redac.CSVRenderer{}
	case "json":
		c.renderer = &redac.JSONRenderer{}
	case "yaml":
		c.renderer = &redac.YAMLRenderer{}
	default:
		return nil, fmt.Errorf("unknown format: %s", formatStr)
	}

	return c, nil
}

func (c *RedacCommand) getConfigContgext(contextName string) (*redac.ConfigContext, error) {
	configCtx := c.config.Contexts[contextName]
	if configCtx == nil {
		return nil, fmt.Errorf("context not found")
	}
	return configCtx, nil
}

func (c *RedacCommand) getRedashClient(configCtx *redac.ConfigContext) (*redac.RedashClient, error) {
	rc, err := redac.NewRedashClient(configCtx.Endpoint, configCtx.APIKey, c.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create redash client: %w", err)
	}

	return rc, nil
}

func (c *RedacCommand) Run(cmd *cobra.Command, args []string) (error, bool) {
	configCtx, err := c.getConfigContgext(c.contextName)
	if err != nil {
		return fmt.Errorf("failed to get config context: %w", err), true
	}
	rc, err := c.getRedashClient(configCtx)
	if err != nil {
		return fmt.Errorf("failed to get redash client: %w", err), true
	}

	params, err := c.query.GetTemplateParams(c.queryArgs)
	if err != nil {
		return fmt.Errorf("failed to get template params: %w", err), true
	}

	result, err := rc.QueryAndWaitResult(c.ctx, redac.RedashPostQueryResultRequest{
		ApplyAutoLimit: !c.noLimit,
		DataSourceID:   configCtx.DataSourceID,
		Parameters:     params,
		Query:          c.query.Data,
	})
	if err != nil {
		return fmt.Errorf("failed to query: %w", err), false
	}

	tableData := result.GetTable()
	c.logger.Debug("render", "table", tableData)
	c.renderer.SetShowHeader(!c.noHeader)
	if err := c.renderer.Render(os.Stdout, tableData); err != nil {
		return fmt.Errorf("failed to render: %w", err), false
	}

	return nil, false
}
