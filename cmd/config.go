package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/matt/sussout/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configUseCmd)
	configCmd.AddCommand(configAddCmd)
	configCmd.AddCommand(configRemoveCmd)
	configCmd.AddCommand(configListCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration and presets",
	Long:  `Show active settings, available presets, and the config file location.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := config.LoadFile()
		if err != nil {
			return err
		}

		cfg := config.Load("")

		fmt.Printf("\nConfig file: %s\n\n", config.ConfigPath())

		fmt.Println("Active settings:")
		fmt.Printf("  %-18s %s\n", "LLM Base URL:", cfg.LLM.BaseURL)
		modelDisplay := cfg.LLM.Model
		if modelDisplay == "" {
			modelDisplay = "(auto-detect)"
		}
		fmt.Printf("  %-18s %s\n", "Model:", modelDisplay)
		if cfg.LLM.APIKey != "" {
			fmt.Printf("  %-18s %s\n", "API Key:", maskKey(cfg.LLM.APIKey))
		} else {
			fmt.Printf("  %-18s (not set)\n", "API Key:")
		}

		fmt.Println("\nPresets:")
		for name, p := range f.Presets {
			marker := " "
			if name == f.DefaultPreset {
				marker = "*"
			}
			modelDisplay := p.Model
			if modelDisplay == "" {
				modelDisplay = "(auto-detect)"
			}
			fmt.Printf("  %s %-17s  %-35s  %s\n", marker, name, p.BaseURL, modelDisplay)
		}
		fmt.Println("\n  * = default")

		fmt.Printf("\nCommands: config use | config add | config remove | config list\n\n")
		return nil
	},
}

var configUseCmd = &cobra.Command{
	Use:   "use <preset>",
	Short: "Set the default preset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		f, err := config.LoadFile()
		if err != nil {
			return err
		}
		if _, ok := f.Presets[name]; !ok {
			return fmt.Errorf("preset '%s' not found. Use 'sussout config list' to see available presets", name)
		}
		f.DefaultPreset = name
		if err := config.SaveFile(f); err != nil {
			return err
		}
		fmt.Printf("Default preset set to '%s'\n", name)
		return nil
	},
}

var configAddCmd = &cobra.Command{
	Use:   "add <name> [url]",
	Short: "Add a new preset, optionally providing the URL",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://") {
			return fmt.Errorf("first argument is the preset name, not a URL.\nUsage: sussout config add <name> [url]\nExample: sussout config add my-server http://192.168.1.187:1234/v1")
		}

		f, err := config.LoadFile()
		if err != nil {
			return err
		}
		if _, ok := f.Presets[name]; ok {
			fmt.Printf("Preset '%s' already exists. Overwrite? [y/N]: ", name)
			reader := bufio.NewReader(os.Stdin)
			resp, _ := reader.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(resp)) != "y" {
				return nil
			}
		}

		reader := bufio.NewReader(os.Stdin)
		var baseURL string

		if len(args) == 2 {
			baseURL = args[1]
		} else {
			fmt.Print("Base URL: ")
			baseURL, _ = reader.ReadString('\n')
			baseURL = strings.TrimSpace(baseURL)
		}

		fmt.Print("Model (press Enter for auto-detect): ")
		model, _ := reader.ReadString('\n')
		model = strings.TrimSpace(model)

		fmt.Print("API Key (optional): ")
		apiKey, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)

		f.Presets[name] = config.Preset{
			BaseURL: baseURL,
			Model:   model,
			APIKey:  apiKey,
		}
		if err := config.SaveFile(f); err != nil {
			return err
		}
		fmt.Printf("Preset '%s' saved.\n", name)
		return nil
	},
}

var configRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a preset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		f, err := config.LoadFile()
		if err != nil {
			return err
		}
		if _, ok := f.Presets[name]; !ok {
			return fmt.Errorf("preset '%s' not found", name)
		}
		if name == f.DefaultPreset {
			return fmt.Errorf("cannot remove the default preset. Use 'sussout config use' first")
		}

		fmt.Printf("Remove preset '%s'? [y/N]: ", name)
		reader := bufio.NewReader(os.Stdin)
		resp, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(resp)) != "y" {
			return nil
		}

		delete(f.Presets, name)
		if err := config.SaveFile(f); err != nil {
			return err
		}
		fmt.Printf("Preset '%s' removed.\n", name)
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all presets with details",
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := config.LoadFile()
		if err != nil {
			return err
		}
		fmt.Println()
		for name, p := range f.Presets {
			marker := " "
			if name == f.DefaultPreset {
				marker = "*"
			}
			fmt.Printf("  %s %s\n", marker, name)
			fmt.Printf("    Base URL: %s\n", p.BaseURL)
			if p.Model != "" {
				fmt.Printf("    Model:    %s\n", p.Model)
			} else {
				fmt.Printf("    Model:    (auto-detect)\n")
			}
			if p.APIKey != "" {
				fmt.Printf("    API Key:  %s\n", maskKey(p.APIKey))
			}
			fmt.Println()
		}
		return nil
	},
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
