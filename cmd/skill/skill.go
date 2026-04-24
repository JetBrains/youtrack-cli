package skill

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tiulpin/instill"

	"github.com/JetBrains/youtrack-cli/internal/output"
)

const defaultSkill = "youtrack-cli"

// NewCmd returns the root `skill` command with subcommands.
// fs is the embedded filesystem containing bundled skill files.
func NewCmd(fs embed.FS) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage AI coding agent skills",
		Long: `Install, update, or remove YouTrack skills for AI coding agents.

Skills teach AI coding agents (Claude Code, Cursor, and others) how
to drive the youtrack CLI effectively. Skills can be installed
globally for the current user or locally into a project.`,
		Args: cobra.NoArgs,
	}
	cmd.AddCommand(newSkillListCmd(fs))
	cmd.AddCommand(newSkillInstallCmd(fs))
	cmd.AddCommand(newSkillUpdateCmd(fs))
	cmd.AddCommand(newSkillRemoveCmd(fs))
	return cmd
}

// SkillInfo is the JSON/table representation of a bundled skill.
type SkillInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Default bool   `json:"default"`
}

type skillOptions struct {
	agents  []string
	all     bool
	project bool
}

func newSkillListCmd(fs embed.FS) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available skills bundled with this release",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			metas := instill.ListSkills(fs)
			if len(metas) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No skills bundled.")
				return nil
			}
			infos := make([]SkillInfo, len(metas))
			for i, m := range metas {
				infos[i] = SkillInfo{Name: m.Name, Version: m.Version, Default: m.Name == defaultSkill}
			}
			format, _ := cmd.Root().PersistentFlags().GetString("format")
			if format == "table" {
				return printSkillTable(cmd.OutOrStdout(), infos)
			}
			return output.Print(cmd, infos)
		},
	}
}

func printSkillTable(w io.Writer, infos []SkillInfo) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer func() { _ = tw.Flush() }()
	_, _ = fmt.Fprintln(tw, "NAME\tVERSION\t")
	for _, s := range infos {
		marker := ""
		if s.Default {
			marker = "(default)"
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\n", s.Name, s.Version, marker)
	}
	return nil
}

func newSkillInstallCmd(fs embed.FS) *cobra.Command {
	opts := &skillOptions{}
	cmd := &cobra.Command{
		Use:   "install [skill-name...]",
		Short: "Install skills for AI coding agents",
		Long: `Install YouTrack skills so AI coding agents can use youtrack commands.

When no skill name is given, installs the default skill (youtrack-cli).
Use --all to install every bundled skill.
Installs globally by default. Use --project to install to the current project only.
Auto-detects installed agents when --agent is not specified.`,
		Example: `  youtrack skill install
  youtrack skill install youtrack-cli
  youtrack skill install --all
  youtrack skill install --agent claude-code
  youtrack skill install --project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillInstall(cmd, fs, opts, args, false)
		},
	}
	addSkillFlags(cmd, opts)
	return cmd
}

func newSkillUpdateCmd(fs embed.FS) *cobra.Command {
	opts := &skillOptions{}
	cmd := &cobra.Command{
		Use:   "update [skill-name...]",
		Short: "Update skills for AI coding agents",
		Long: `Update YouTrack skills to the latest version bundled with this release.

When no skill name is given, updates the default skill (youtrack-cli).
Use --all to update every bundled skill.
Skips if the installed version already matches.
Auto-detects installed agents when --agent is not specified.`,
		Example: `  youtrack skill update
  youtrack skill update --all
  youtrack skill update --agent claude-code
  youtrack skill update --project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillInstall(cmd, fs, opts, args, true)
		},
	}
	addSkillFlags(cmd, opts)
	return cmd
}

func newSkillRemoveCmd(fs embed.FS) *cobra.Command {
	opts := &skillOptions{}
	cmd := &cobra.Command{
		Use:   "remove [skill-name...]",
		Short: "Remove skills from AI coding agents",
		Long: `Remove YouTrack skills from AI coding agents.

When no skill name is given, removes the default skill (youtrack-cli).
Use --all to remove every bundled skill.`,
		Example: `  youtrack skill remove
  youtrack skill remove --all
  youtrack skill remove --agent claude-code
  youtrack skill remove --project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillRemove(cmd, fs, opts, args)
		},
	}
	addSkillFlags(cmd, opts)
	return cmd
}

func addSkillFlags(cmd *cobra.Command, opts *skillOptions) {
	cmd.Flags().StringSliceVarP(&opts.agents, "agent", "a", nil, "Target agent(s); auto-detects if omitted")
	cmd.Flags().BoolVar(&opts.all, "all", false, "Install/update all bundled skills")
	cmd.Flags().BoolVar(&opts.project, "project", false, "Install to current project instead of globally")
}

func runSkillInstall(cmd *cobra.Command, fs embed.FS, opts *skillOptions, args []string, checkVersion bool) error {
	agents, err := resolveAgents(cmd, opts)
	if err != nil {
		return err
	}
	names, err := resolveNames(fs, opts, args)
	if err != nil {
		return err
	}
	versions := bundledVersions(fs)
	for _, name := range names {
		if _, ok := versions[name]; !ok {
			return fmt.Errorf("unknown skill %q; run 'youtrack skill list' to see available skills", name)
		}
	}
	instillOpts := instill.Options{
		Agents:     agents,
		Skills:     names,
		ProjectDir: ".",
		Global:     !opts.project,
	}
	if checkVersion {
		allUpToDate := true
		for _, name := range names {
			installed, err := instill.InstalledVersion(name, instillOpts)
			if err != nil {
				return err
			}
			if installed != versions[name] {
				allUpToDate = false
				break
			}
		}
		if allUpToDate {
			for _, name := range names {
				output.PrintSuccess(cmd, fmt.Sprintf("skill %s already up to date (%s)", name, versions[name]))
			}
			return nil
		}
	}
	results, err := instill.Install(fs, instillOpts)
	if err != nil {
		return err
	}
	for _, r := range results {
		bundled := versions[r.Skill]
		switch {
		case !r.Existed:
			output.PrintSuccess(cmd, fmt.Sprintf("installed %s for %s (%s)", r.Skill, r.Agent, bundled))
		case r.PriorVersion != "" && r.PriorVersion != bundled:
			output.PrintSuccess(cmd, fmt.Sprintf("updated %s for %s (%s → %s)", r.Skill, r.Agent, r.PriorVersion, bundled))
		case r.PriorVersion == "":
			output.PrintSuccess(cmd, fmt.Sprintf("updated %s for %s (unversioned → %s)", r.Skill, r.Agent, bundled))
		default:
			output.PrintSuccess(cmd, fmt.Sprintf("reinstalled %s for %s (%s)", r.Skill, r.Agent, bundled))
		}
	}
	return nil
}

func runSkillRemove(cmd *cobra.Command, fs embed.FS, opts *skillOptions, args []string) error {
	agents, err := resolveAgents(cmd, opts)
	if err != nil {
		return err
	}
	names, err := resolveNames(fs, opts, args)
	if err != nil {
		return err
	}
	versions := bundledVersions(fs)
	for _, name := range names {
		if _, ok := versions[name]; !ok {
			return fmt.Errorf("unknown skill %q; run 'youtrack skill list' to see available skills", name)
		}
	}
	for _, name := range names {
		results, err := instill.Remove(name, instill.Options{
			Agents:     agents,
			ProjectDir: ".",
			Global:     !opts.project,
		})
		if err != nil {
			return err
		}
		for _, r := range results {
			if r.Existed {
				output.PrintSuccess(cmd, fmt.Sprintf("removed %s from %s", name, r.Agent))
			} else {
				output.PrintWarn(cmd, fmt.Sprintf("skill %s not installed for %s, nothing to remove", name, r.Agent))
			}
		}
	}
	return nil
}

func resolveNames(fs embed.FS, opts *skillOptions, args []string) ([]string, error) {
	if opts.all && len(args) > 0 {
		return nil, errors.New("cannot specify both --all and skill names")
	}
	if opts.all {
		skills := instill.ListSkills(fs)
		names := make([]string, len(skills))
		for i, s := range skills {
			names[i] = s.Name
		}
		return names, nil
	}
	if len(args) == 0 {
		return []string{defaultSkill}, nil
	}
	return args, nil
}

func bundledVersions(fs embed.FS) map[string]string {
	skills := instill.ListSkills(fs)
	m := make(map[string]string, len(skills))
	for _, s := range skills {
		m[s.Name] = s.Version
	}
	return m
}

func resolveAgents(cmd *cobra.Command, opts *skillOptions) ([]string, error) {
	if len(opts.agents) > 0 {
		return opts.agents, nil
	}
	detected, err := instill.Detect(".", !opts.project)
	if err != nil {
		return nil, fmt.Errorf("detecting agents: %w", err)
	}
	if len(detected) == 0 {
		names := instill.AgentNames()
		var hint string
		if len(names) > 5 {
			hint = fmt.Sprintf("%s, %s, %s, ... (%d total)", names[0], names[1], names[2], len(names))
		} else {
			hint = fmt.Sprintf("%v", names)
		}
		return nil, fmt.Errorf("no AI coding agents detected; use --agent to specify one (available: %s)", hint)
	}
	result := make([]string, len(detected))
	for i, a := range detected {
		result[i] = a.Name
	}
	return result, nil
}
