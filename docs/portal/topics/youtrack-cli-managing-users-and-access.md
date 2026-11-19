# Managing Users and Access

<show-structure for="chapter" depth="2"/>

YouTrack CLI manages users, groups, roles, and organizations from the command line. Use these commands to create users, control who can sign in, and grant the permissions that roles carry. Most commands that grant access take opaque IDs, so look up the ID before you run them.

## Each command expects a login, a name, or an opaque ID

Different commands expect different identifiers. Most user commands accept a login. Groups are addressed by name for basic operations, but some subcommands need the opaque ID that the API returns. Roles, permissions, projects in `--project`, and organizations are always addressed by their opaque IDs.

<table style="none">
<tr><td>

User login
</td><td>

Most user commands accept a login, for example `jane.doe`, or the opaque user ID. This applies to `user view`, `user ban`, `user unban`, `user delete`, `user profile`, `user locale`, `user timezone`, and `user tag list`.
</td></tr>
<tr><td>

Group name
</td><td>

`group view` and `group delete` accept a group name, for example `Developers`, or the opaque group ID.
</td></tr>
<tr><td>

Group ID
</td><td>

`group member list`, `group role list`, `group role add`, and the `--group`/`--holder` flags require the opaque group ID, for example `3-2`.
</td></tr>
<tr><td>

Role and permission ID
</td><td>

`role view`, `role update`, `role delete`, `role permission`, and `--role` require opaque IDs. Get role IDs from `youtrack role list` and permission IDs from `youtrack role permissions`.
</td></tr>
<tr><td>

Project ID
</td><td>

The `--project` flag of `role grant` and `group role add` requires the opaque project ID, for example `0-1`, not the short name `DEMO`. Get it from `youtrack project view DEMO --json`.
</td></tr>
<tr><td>

Organization ID
</td><td>

The `--org` flag and `org delete` require the opaque organization ID. Get it from `youtrack org list --json`.
</td></tr>
</table>

To look up an opaque ID, run the matching `view` or `list` command and read the `id` field:

```Shell
youtrack user view jane.doe --json
youtrack group view Developers --json
youtrack role list --json
youtrack project view DEMO --json
```

<warning>

The `--role`, `--user`, and `--group` flags on `role grant` and the `<group-id>` argument of the `group member` and `group role` commands take opaque IDs, not logins or names. Retrieve the ID first, then pass it. See [Scripting and Automation](youtrack-cli-scripting.md) for extracting a single field.

</warning>

## List users with a query, up to 100 by default

List users with `youtrack user list`. Narrow the result with `--query`. The command returns at most 100 users. If you need more, raise `--limit`. To show the currently authenticated user, run `youtrack user current`:

```Shell
youtrack user list --query "banned: false"
youtrack user list --limit 500
youtrack user current
```

## View a user by login

View one user by login:

```Shell
youtrack user view jane.doe
```

## A new user needs a login, name, email, and password

<warning>

Do not type the password as a literal. A literal password stays in your shell history. Pass it through an environment variable or your shell's secret handling.

</warning>

Create a user with `youtrack user create`. The login, full name, email, and password are all required:

```Shell
youtrack user create --login jane.doe --fullName "Jane Doe" --email jane.doe@example.com --password "$NEW_USER_PASSWORD"
```

YouTrack returns the new user, including its opaque `id`. By default the user type is `standard`. To create another user type, add `--type agent` or `--type reporter`.

## Ban a user to block sign-in and keep the history

Ban a user to block sign-in and keep the user's history:

```Shell
youtrack user ban jane.doe
youtrack user unban jane.doe
```

`user ban` asks for confirmation. Add `--force` to skip the prompt in scripts.

## Deleting a user is permanent and needs a successor

<warning>

If you only need to revoke access, ban the user instead. You can restore a banned user with `youtrack user unban`. You cannot restore a deleted user.

</warning>

YouTrack reassigns the deleted user's references — assignments, comments, and authored issues — to the successor:

```Shell
youtrack user delete jane.doe --successor john.doe
```

The command asks for confirmation. Add `--force` to skip the prompt in scripts. You cannot delete the currently authenticated user, and the successor must be a different user.

## User preferences, drafts, and tags have their own subcommands

View a user's general profile, the time zone, and set the locale and time zone:

```Shell
youtrack user profile view jane.doe
youtrack user timezone view jane.doe
youtrack user locale set jane.doe en_US
youtrack user timezone set jane.doe Europe/Berlin
```

The other `user profile` subcommands change one profile section each. Run `youtrack user profile <subcommand> --help` for the arguments.

<table>
<tr><td>Command</td><td>Description</td></tr>
<tr><td>

`user profile appearance`, `articles`, `general`, `grazie`, `issues-list`, `notifications`

</td><td>

Set one profile section. Pass the settings as a JSON body with `--body`.

</td></tr>
<tr><td><code>user profile property &lt;user&gt; &lt;key&gt; &lt;value&gt;</code>

</td><td>Set one profile property.

</td></tr>
<tr><td><code>user profile tips &lt;user&gt; &lt;key=true|false&gt;...</code>

</td><td>Set profile tip flags.

</td></tr>
<tr><td><code>user profile timetracking view &lt;user&gt;</code>

</td><td>View the time tracking profile.

</td></tr>
<tr><td>

`user drafts issues`, `user drafts articles`

</td><td>

List your issue or article drafts. Drafts are private to their owner, so the commands default to `me`.

</td></tr>
<tr><td>

`user drafts delete me`

</td><td>

Delete all your issue drafts. The command asks for confirmation. Add `--force` to skip the prompt.

</td></tr>
<tr><td><code>user tag list &lt;user&gt;</code>

</td><td>List the tags that a user owns.

</td></tr>
</table>

## Groups are created by name

Create, list, view, and delete groups. Narrow `group list` with `--query`:

```Shell
youtrack group create --name "QA Team"
youtrack group list
youtrack group list --query "QA"
youtrack group view "QA Team"
```

<warning>

The `--successor` group inherits the references of the deleted group. A broad successor group can expand access to the reassigned resources. The CLI does not select a default successor, so choose it explicitly.

</warning>

Delete a group and reassign its references to a successor group. The `--successor` flag is required and takes a group name or ID:

```Shell
youtrack group delete "QA Team" --successor Developers
```

`group delete` asks for confirmation. Add `--force` to skip the prompt in scripts.

### Group member commands take opaque IDs

List, add, or remove members. Both the group and user arguments are opaque IDs:

```Shell
youtrack group member list 3-2
youtrack group member add 3-2 1-5
youtrack group member remove 3-2 1-5 --force
```

Resolve IDs first with `group view <name> --json` and `user view <login> --json`.

### A role granted to a group applies to every member

Grant a role to every member of a group. Pass the opaque group ID as the argument and the opaque role ID through `--role`. If you want to scope the role to one project, add `--project` with the opaque project ID. If you want to scope it to one organization, add `--org`. If you omit both, the assignment is global:

```Shell
youtrack group role add 3-2 --role 2-3
youtrack group role add 3-2 --role 2-3 --project 0-1
youtrack group role add 3-2 --role 2-3 --org 0-0
youtrack group role list 3-2
```

`--project` and `--org` are mutually exclusive. These two commands are aliases: `group role add` is `role grant --group`, and `group role list` is `role assignments --holder`. Use whichever reads better in context.

## A role bundles permissions

A role bundles permissions. List existing roles and the permissions you can attach:

```Shell
youtrack role list
youtrack role permissions
```

<warning>

`role update` replaces the full permission set on the role. Permissions that you do not list are removed. If you need to change one permission, use `role permission add` or `role permission remove`. You cannot undo `role delete`.

</warning>

View, create, update, and delete roles by opaque ID:

```Shell
youtrack role view 2-3
youtrack role create --name "Release Manager" --description "Ships releases" --permissions "<permId1>,<permId2>"
youtrack role update 2-3 --permissions "<permId1>,<permId2>"
youtrack role delete 2-3
```

The `--permissions` flag takes a comma-separated list of permission IDs. Run `youtrack role permissions` to look them up. `--description` is optional.

Add or remove a single permission on an existing role without resetting the whole set:

```Shell
youtrack role permission list 2-3
youtrack role permission add 2-3 <permId>
youtrack role permission remove 2-3 <permId>
```

`role update`, `role delete`, and `role permission remove` ask for confirmation. Add `--force` to skip the prompt in scripts.

## Grants use holder IDs, revokes use assignment IDs

Grant a role to a user or a group with `youtrack role grant`. Provide the role ID and exactly one of `--user` or `--group`, both opaque IDs. The grant is global unless you narrow it: `--project` scopes it to a single project, `--org` to a single organization. Both flags take opaque IDs and are mutually exclusive:

```Shell
youtrack role grant --role 2-3 --user 1-5
youtrack role grant --role 2-3 --group 3-2 --project 0-1
youtrack role grant --role 2-3 --group 3-2 --org 0-0
```

Get project IDs from `youtrack project view <shortName> --json` and organization IDs from `youtrack org list --json`.

Revoking uses the assignment's own ID, not the user or role ID. Find it first with `role assignments`, then pass it to `role revoke`:

```Shell
youtrack role revoke <assignedRoleID>
```

`role revoke` asks for confirmation. Add `--force` to skip the prompt in scripts.

### Role assignments list the IDs that revoke needs

List every role assignment, or filter to one holder with `--holder`. A holder is any principal a role can be granted to — a user, group, team, or organization — so `--holder` accepts an opaque ID of any of those types. The `--user` and `--group` flags are readability aliases for `--holder`; YouTrack does not distinguish them:

```Shell
youtrack role assignments
youtrack role assignments --holder 1-5
```

The `id` field of each row is the `<assignedRoleID>` you pass to `role revoke`.

## Organizations are deleted by opaque ID

<warning>

`org delete` removes the organization permanently. The command asks for confirmation. Add `--force` only in scripts that you have tested.

</warning>

List, create, and delete organizations. `org delete` takes the organization's opaque ID:

```Shell
youtrack org list
youtrack org create --name "Acme Inc"
youtrack org delete <id>
```

<seealso>
<category ref="04">
<a href="youtrack-cli-managing-projects.md">Managing Projects</a>
<a href="youtrack-cli-administration.md">Administration</a>
<a href="youtrack-cli-scripting.md">Scripting and Automation</a>
</category>
<category ref="06">
<a href="youtrack-cli.md">YouTrack CLI</a>
<a href="youtrack-cli-authentication.md">Authentication</a>
<a href="youtrack-cli-configuration.md">Configuration</a>
</category>
</seealso>
