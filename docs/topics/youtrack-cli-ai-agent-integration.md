[//]: # (title: AI Agent Integration)

<show-structure for="chapter" depth="2"/>

YouTrack CLI bundles a skill that teaches artificial intelligence (AI) coding agents how to use the `youtrack` command. Install the skill once with `youtrack skill install`. After that, your agent knows the commands, the output formats, and the YouTrack-specific gotchas, and you do not paste instructions into every session.

The skill uses the standard Agent Skills format, so it works with any compatible agent, for example Claude Code, Cursor, Codex, Gemini CLI, and GitHub Copilot.

## List the bundled skills

List the skills bundled with your `youtrack` release, with each one's version and which is installed by default:

```Shell
youtrack skill list
```

The output marks the default skill:

```
NAME          VERSION
youtrack-cli  2.1.0    (default)
```

`youtrack-cli` is the default skill. Commands that take a skill name install it when you pass no name.

## Install the skill as a stub that reads the live guide

Install the skill for all detected AI coding agents:

```Shell
youtrack skill install
```

The command detects which AI coding agents are set up on your system and installs the skill for each one. Detection uses each agent's configuration directory. If the command detects no agent, it reports an error and asks you to name one with `--agent`.

Each install is a short stub. The stub tells the agent to run `youtrack skill guide`, which prints the full guide from the installed `youtrack` binary. So the guide always matches your CLI version, also after you upgrade the CLI.

> The skill installs globally by default, into your home configuration. To install it for one project only, use `--project`.
>
{style="note"}

### Install for specific agents

Target one or more agents by their identifier. Repeat `--agent` to name several:

```Shell
youtrack skill install --agent claude-code
youtrack skill install --agent claude-code --agent cursor
```

Agent identifiers are lowercase, for example `claude-code`, `cursor`, `codex`, `gemini-cli`, and `github-copilot`. If you pass an unknown identifier, the CLI reports the name it did not recognize.

### Install at the project level

Install the skill into the current project instead of globally:

```Shell
youtrack skill install --project
```

A project-level install places the skill under the project directory, so it travels with the repository and applies only when an agent works in that project.

### Install every bundled skill

By default only the `youtrack-cli` skill is installed. Use `--all` to install every skill bundled with the release:

```Shell
youtrack skill install --all
```

## Read the guide with `skill guide` {id="guide"}

Print the guide of a bundled skill as Markdown. Agents run this command when they read the installed stub, and you can run it yourself:

```Shell
youtrack skill guide
youtrack skill guide youtrack-cli gotchas
youtrack skill guide youtrack-cli references/commands
```

The command takes two optional arguments: `[skill] [topic]`. Without a skill, it uses the default skill `youtrack-cli`. Without a topic, it prints the overview and the list of topics. With `--json`, it prints the Markdown as a JSON string.

## Update the stub with `skill update`

Update the installed stub and its version to the ones bundled with your current `youtrack` release:

```Shell
youtrack skill update
```

The guide content comes from `skill guide`, so you do not need `skill update` to get a newer guide. The command skips the update when the installed version already matches the bundled version. To target specific agents or the project level, use the same flags as `install`:

```Shell
youtrack skill update --agent claude-code
youtrack skill update --project
```

## Remove the skill

Remove the skill from your AI coding agents:

```Shell
youtrack skill remove
```

To target specific agents or the project level, use the same flags:

```Shell
youtrack skill remove --agent claude-code
youtrack skill remove --project
```

## Flags shared by install, update, and remove

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

## What the skill teaches an agent

The skill gives an agent a task-oriented reference to YouTrack CLI. It covers:

- Verifying authentication with `youtrack user current` before doing work.
- Finding and creating issues, and applying commands with `youtrack issue command`.
- Managing projects, users, groups, roles, boards, articles, and tags.
- Output formats, including when to add `--json` for machine-readable output.
- YouTrack-specific gotchas, such as `project create` requiring `--leader`, `project team add-user` expecting opaque IDs rather than logins, and how logins are wrapped for the command parser.

## Read-only mode blocks accidental writes, not a determined agent {id="read-only"}

> Read-only mode is a safety net, not a security boundary. `YOUTRACK_RO` is an environment variable, so the agent can unset it, and the token keeps its full access. To limit an agent, give it a token from a user that has only the permissions the agent needs. See [Security](youtrack-cli-security.md).
>
{style="warning"}

If you want to prevent accidental creates, updates, and deletes, export `YOUTRACK_RO=1` in the agent's session before the agent runs any `youtrack` command. Only the value `1` enables read-only mode. Other values, such as `true` or `yes`, do not block writes.

```Shell
export YOUTRACK_RO=1
```

In read-only mode, the agent can list issues, run search queries, and read any entity. Every command that writes data fails:

```
youtrack issue create ...
# error: read-only mode is enabled (YOUTRACK_RO=1): write operations are blocked
```

Two commands still work, because YouTrack exposes them as `POST` requests that only compute a result: `issue count` and `issue command suggest`.

## Agents get JSON output without flags

Agents read structured output. YouTrack CLI enables the `--json` flag automatically when stdout is not a terminal, so commands piped or captured by an agent return JSON without any extra flag. See [Scripting and Automation](youtrack-cli-scripting.md) for output shaping with `--json`, `--fields`, and `--plain`.

<seealso>
    <category ref="06">
        <a href="youtrack-cli-scripting.md">Scripting and Automation</a>
        <a href="youtrack-cli-configuration.md">Configuration</a>
        <a href="youtrack-cli-security.md">Security</a>
    </category>
    <category ref="04">
        <a href="youtrack-cli-get-started.md">Getting Started</a>
        <a href="youtrack-cli-authentication.md">Authentication</a>
    </category>
</seealso>
