//go:generate yaegi extract -name main -include "^Connection,^Verdict" github.com/safing/portmaster/plugin/shared/proto
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/safing/portmaster/plugin/framework"
	cmd "github.com/safing/portmaster/plugin/framework/cmds"
	"github.com/safing/portmaster/plugin/shared"
	"github.com/safing/portmaster/plugin/shared/proto"
	"github.com/spf13/cobra"
)

type Config struct {
	Paths []string `json:"paths"`
}

func registerAndWatchOption(ctx context.Context, engine *RuleEngine) error {
	err := framework.Config().RegisterOption(ctx, &proto.Option{
		Name:        "Rule Directories",
		Description: "A list of directory to load rules from",
		Key:         "ruleDirectories",
		Default: &proto.Value{
			StringArray: []string{
				filepath.Join(framework.BaseDirectory(), "yaegi-rules"),
			},
		},
		OptionType: proto.OptionType_OPTION_TYPE_STRING_ARRAY,
	})
	if err != nil {
		return err
	}

	ch, err := framework.Config().WatchValue(framework.Context(), "ruleDirectories")
	if err != nil {
		return err
	}

	go func() {
		for msg := range ch {
			log.Println("[INFO] received new value for ruleDirectories, resetting engine")

			engine.Reset()
			if err := engine.LoadPaths(msg.GetValue().StringArray...); err != nil {
				log.Printf("[ERROR] failed to load rules: %s", err)
			}
		}
	}()

	return nil
}

func main() {
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	root := &cobra.Command{
		Use: filepath.Base(execPath),
		Run: func(cmd *cobra.Command, args []string) {
			engine := NewRuleEngine()

            if err := framework.RegisterDecider(engine); err != nil {
                panic(err)
            }

			framework.OnInit(func(ctx context.Context) error {
				var cfg Config

				if err := framework.ParseStaticConfig(&cfg); err != nil {
					if errors.Is(err, framework.ErrNoStaticConfig) {
						// there's no static configuration so let's register
						// a configuration option in the Portmaster instead.
						if err := registerAndWatchOption(ctx, engine); err != nil {
							return err
						}

					} else {
						return err
					}
				}

				return engine.LoadPaths(cfg.Paths...)
			})

			framework.Serve()
		},
	}

	cfg := &cmd.InstallCommandConfig{
		PluginName: "portmaster-plugin-yaegi",
		Types: []shared.PluginType{
			shared.PluginTypeDecider,
		},
	}

	installCmd := cmd.InstallCommand(cfg)
	var ruleDirs []string

	flags := installCmd.Flags()
	{
		flags.StringSliceVar(&ruleDirs, "rules", nil, "Path to rule list directory")
	}

	installCmd.PreRun = func(cmd *cobra.Command, args []string) {
		if len(ruleDirs) > 0 {
			blob, err := json.MarshalIndent(Config{
				Paths: ruleDirs,
			}, "", "  ")
			if err != nil {
				log.Fatal(err)
			}

			cfg.StaticConfig = blob
		}
	}

	root.AddCommand(
		installCmd,
		testLoadRulesCommand(),
	)

	if err := root.Execute(); err != nil {
		log.Fatal(err)
	}
}

func testLoadRulesCommand() *cobra.Command {
	return &cobra.Command{
		Use:  "test [rule-dir...]",
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			engine := NewRuleEngine()

			if err := engine.LoadPaths(args...); err != nil {
				log.Fatal(err)
			}
		},
	}
}
