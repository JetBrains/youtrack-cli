# Administration
<show-structure for="chapter" depth="2"/>

YouTrack CLI exposes instance-wide administration: global settings, custom fields, workflows, and app integrations. This page covers the commands under `settings`, `field`, `workflow`, and `app`. To manage users, groups, roles, and organizations, see [Managing Users and Access](youtrack-cli-managing-users-and-access.md).

<note>

These operations act on the whole instance, not a single project. You need administrator permissions in YouTrack to run most of them, and the CLI returns a permission error otherwise.

</note>

## Global settings are read-only, feature flags are not

Print the instance's global settings with `youtrack settings view` (alias `settings get`):

```Shell
youtrack settings view
```

<note>

No endpoint lists the available flag IDs. Get them from JetBrains support, the release notes, or the administration UI before you run `settings feature`. The command asks for confirmation before it changes a flag. Add `--force` to skip the prompt.

</note>

`settings view` is read-only. To turn a feature flag on or off, pass its flag ID to `settings feature`:

```Shell
youtrack settings feature --name <flag-id> --enabled=false
```

## Custom fields are defined once for the whole instance

List every custom field defined on the instance:

```Shell
youtrack field list
```

<warning>

`field delete` removes the field from every project that uses it. The CLI asks for confirmation first. Add `--force` to skip the prompt.

</warning>

Create a custom field with a name and a type, both required. Delete one by its ID:

```Shell
youtrack field create --name "Severity" --type "enum[1]"
youtrack field delete 58-3
```

The `--type` value is a field type ID. Common values are `string`, `text`, `integer`, `float`, `date`, `enum[1]`, `enum[*]`, `state[1]`, `user[1]`, and `user[*]`. A `[1]` suffix means single-value; `[*]` means multi-value. Run `youtrack field create --help` for the full list. Quote any type that contains brackets so your shell does not expand it.

To inspect the built-in field templates before you create one, run `youtrack field prototypes`.

`field privacy` makes a field private by default. To make the field public again, pass `--private=false`. The command asks for confirmation. Add `--force` to skip the prompt:

```Shell
youtrack field privacy 58-3
youtrack field privacy 58-3 --private=false
```

### Enum, state, and version fields take values from a bundle

Enum, state, and version fields draw their values from a bundle. List bundles of a type, create one, add values, and attach it to a field:

```Shell
youtrack field bundle list --type EnumBundle
youtrack field bundle create --name "Severity values" --type EnumBundle
youtrack field bundle add-value --id 60-1 --value "Blocker" --type EnumBundle
youtrack field bundle attach --field 58-3 --id 60-1 --type EnumBundle
```

The `--type` flag defaults to `EnumBundle` on each of these commands. For other field types, pass the matching bundle type: `EnumBundle`, `StateBundle`, `VersionBundle`, `BuildBundle`, `OwnedBundle`, or `UserBundle`. The short forms `enum`, `state`, `version`, `build`, `owned`, and `user` also work.

### Work item types and work time configure time tracking

List or add the work item types used for time tracking, and view or set the work-time presentation:

```Shell
youtrack field workitem list
youtrack field workitem add-type --name "Development" --auto-attach
youtrack field worktime view
youtrack field worktime set --minutes-a-day 8h
```

## Workflow rules are JavaScript that YouTrack compiles on save

List the workflows on the instance, or only those whose name contains some text (case-insensitive), create one, and delete it by ID:

```Shell
youtrack workflow list
youtrack workflow list --name escalation
youtrack workflow create --name custom-escalation --title "Escalation rules"
youtrack workflow delete 8-2
```

`workflow delete` prompts for confirmation; pass `--force` to skip it. Add a rule to an existing workflow by passing the workflow ID and the rule's JavaScript source. Rule names may contain only lowercase letters, digits, `-`, `_`, `.`, and `~`. YouTrack loads the script when you save it and rejects the rule if the script does not compile:

```Shell
youtrack workflow rule create 8-2 --name on-state-change --script-file on-state-change.js
```

The `--type` flag sets the `$type` that the CLI sends, and defaults to `WorkflowRule`. YouTrack determines the resulting rule type from the script.

View a rule, including its script, and replace the script. Pass `-` to `--script-file` to read the script from stdin:

```Shell
youtrack workflow rule view 8-2 9-14
youtrack workflow rule view 8-2 9-14 --json | jq -r .script > on-state-change.js
youtrack workflow rule update 8-2 9-14 --script-file on-state-change.js
```

`workflow rule update` reads the rule back after the write and compares the stored script with the one it sent, byte for byte. When they differ, it reports the sent and stored byte counts and exits with a non-zero status. Verification confirms the stored script, not its bindings in each project; project requirements can still fail. A rule is shared by every project the workflow is attached to, so a script YouTrack rejects or changes affects all of them.

<note>

Table and plain output strip control characters such as carriage returns from the script. Use `--json` when you need the exact stored source.

</note>

<warning>

`workflow restore` runs at once. It has no confirmation prompt and no `--force` flag. The command affects instance-wide automation, so review your workflows before you run it.

</warning>

To restore the default workflows, run `youtrack workflow restore`:

```Shell
youtrack workflow restore
```

## Apps are installed in the web UI and managed here

### Apps

<warning>

`app deactivate` runs at once, with no confirmation prompt. `app delete` and `app plugin delete` ask for confirmation. Add `--force` to skip the prompt.

</warning>

List the installed apps and their plugins, and deactivate or remove an app by ID:

```Shell
youtrack app list
youtrack app plugin list
youtrack app deactivate 3-7
youtrack app delete 3-7
```

Remove a single plugin with `youtrack app plugin delete <id>`.

<note>

The CLI manages apps that are already installed. Install a new app from the YouTrack web interface, then list, deactivate, or delete it here.

</note>

## Every command on this page supports JSON output

Every command on this page returns structured data. Add `-j` (or `--json`) for JSON, and `-f` to select fields. Field selection helps when you capture an opaque ID to pass to the next command:

```Shell
youtrack field list -j -f id,name,fieldType.id
```

See [Scripting and Automation](youtrack-cli-scripting.md) for piping JSON into `jq` and building multi-step admin scripts.

<seealso>
<category ref="04">
<a href="youtrack-cli-managing-projects.md">Managing Projects</a>
<a href="youtrack-cli-managing-users-and-access.md">Managing Users and Access</a>
<a href="youtrack-cli-scripting.md">Scripting and Automation</a>
</category>
<category ref="06">
<a href="youtrack-cli-configuration.md">Configuration</a>
<a href="youtrack-cli-authentication.md">Authentication</a>
</category>
</seealso>
