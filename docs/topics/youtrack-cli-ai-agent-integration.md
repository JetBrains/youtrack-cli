[//]: # (title: AI Agent Integration)

<show-structure for="chapter" depth="2"/>

YouTrack CLI bundles a skill that teaches AI coding agents how to drive the `youtrack` command. The skill is packaged in the standard Agent Skills format, so it works with any compatible agent, for example Claude Code, Cursor, Codex, Gemini CLI, and GitHub Copilot. Once installed, your agent knows the command surface, output formats, and YouTrack-specific gotchas without you pasting instructions into every session.

## Listing skills

List the skills bundled with your `youtrack` release, with each one's version and which is installed by default:

```Shell
youtrack skill list
```

The output marks the default skill:

```
NAME          VERSION
youtrack-cli  2.0.0    (default)
```

`youtrack-cli` is the default skill. Commands that take a skill name install it when you pass no name.

## Installing the skill

Install the skill for all detected AI agents:

```Shell
youtrack skill install
```

The command auto-detects which AI coding agents are set up on your system and installs the skill for each one. Detection is based on each agent's configuration directory. If no agent is detected, the command reports an error and asks you to name one with `--agent`.

> The skill installs globally by default, into your home configuration. Run `youtrack skill update` after upgrading the CLI to pick up a newer bundled version.
>
{style="note"}

### Installing for specific agents

Target one or more agents by their identifier. Repeat `--agent` to name several:

```Shell
youtrack skill install --agent claude-code
youtrack skill install --agent claude-code --agent cursor
```

Agent identifiers are lowercase, for example `claude-code`, `cursor`, `codex`, `gemini-cli`, and `github-copilot`. If you pass an unknown identifier, the CLI reports the name it did not recognize.

### Installing at the project level

Install the skill into the current project instead of globally:

```Shell
youtrack skill install --project
```

A project-level install places the skill under the project directory, so it travels with the repository and applies only when an agent works in that project.

### Installing every bundled skill

By default only the `youtrack-cli` skill is installed. Use `--all` to install every skill bundled with the release:

```Shell
youtrack skill install --all
```

## Updating the skill

Update the skill to the version bundled with your current `youtrack` release:

```Shell
youtrack skill update
```

The command skips the update when the installed version already matches the bundled version. Target specific agents or update at the project level with the same flags as `install`:

```Shell
youtrack skill update --agent claude-code
youtrack skill update --project
```

## Removing the skill

Remove the skill from your AI coding agents:

```Shell
youtrack skill remove
```

Target specific agents or remove from the project level:

```Shell
youtrack skill remove --agent claude-code
youtrack skill remove --project
```

## skill flags

These flags are shared across `skill install`, `skill update`, and `skill remove`:

<table>
<tr>
<td>

Flag

</td>
<td>

Description

</td>
</tr>
<tr>
<td>

`-a`, `--agent`

</td>
<td>

Target agent by its identifier, for example `claude-code`. Repeat to target several. Auto-detects installed agents when omitted.

</td>
</tr>
<tr>
<td>

`--project`

</td>
<td>

Install into the current project instead of globally.

</td>
</tr>
<tr>
<td>

`--all`

</td>
<td>

Apply to every bundled skill instead of only the default.

</td>
</tr>
</table>

## What the skill teaches

The skill gives an agent a task-oriented reference to YouTrack CLI. It covers:

- Verifying authentication with `youtrack user current` before doing work.
- Finding and creating issues, and applying commands with `youtrack issue command`.
- Managing projects, users, groups, roles, boards, articles, and tags.
- Output formats, including when to add `--json` for machine-readable output.
- YouTrack-specific gotchas, such as `project create` requiring `--leader`, `project team add-user` expecting opaque IDs rather than logins, and how logins are wrapped for the command parser.

## Restricting agents to read-only mode

When you give an agent access to YouTrack, you may want to prevent it from creating, updating, or deleting anything. Set `YOUTRACK_RO=1` to block all write operations:

```Shell
export YOUTRACK_RO=1
```

In read-only mode the agent can list issues, run search queries, and read any entity, but any command that would write data fails:

```
youtrack issue create ...
# error: read-only mode is enabled (YOUTRACK_RO=1): write operations are blocked
```

> `YOUTRACK_RO` is read from the environment. Export it in the agent's session before running any `youtrack` command that could modify data.
>
{style="warning"}

## Getting JSON output for agents

Agents consume structured output. YouTrack CLI enables the `--json` flag automatically when stdout is not a terminal, so commands piped or captured by an agent return JSON without any extra flag. See [Scripting and Automation](youtrack-cli-scripting.md) for output shaping with `--json`, `--fields`, and `--plain`.

<seealso>
    <category ref="reference">
        <a href="youtrack-cli-scripting.md">Scripting and Automation</a>
        <a href="youtrack-cli-configuration.md">Configuration</a>
    </category>
    <category ref="user-guide">
        <a href="youtrack-cli-get-started.md">Getting Started</a>
        <a href="youtrack-cli-authentication.md">Authentication</a>
    </category>
</seealso>
