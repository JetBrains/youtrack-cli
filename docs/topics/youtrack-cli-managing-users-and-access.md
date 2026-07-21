[//]: # (title: Managing Users and Access)

<show-structure for="chapter" depth="2"/>

YouTrack CLI manages users, groups, roles, and organizations from the command line. Use these commands to create accounts, control who can sign in, and grant the permissions that roles carry.

## Identifying users, groups, and roles {id="identifiers"}

Different commands expect different identifiers. Most user commands accept a login. Groups are addressed by name for basic operations, but some subcommands need the opaque ID that the API returns. Roles and permissions are always addressed by their opaque IDs.

<deflist type="medium">
<def title="User login">

Most user commands accept a login, for example `jane.doe`. This applies to `user view`, `user ban`, `user delete`, `user profile`, `user locale`, and `user timezone`.
</def>
<def title="Group name">

`group view` and `group delete` accept a group name, for example `Developers`.
</def>
<def title="Group ID">

`group member list`, `group role list`, `group role add`, and the `--group`/`--holder` flags require the opaque group ID, for example `2-9`.
</def>
<def title="Role and permission ID">

`role view`, `role update`, `role delete`, `role permission`, and `--role` require opaque IDs. Get role IDs from `youtrack role list` and permission IDs from `youtrack role permissions`.
</def>
</deflist>

To look up an opaque ID, run the matching `view` or `list` command and read the `id` field:

```Shell
youtrack user view jane.doe --json
youtrack group view Developers --json
youtrack role list --json
```

> The `--role`, `--user`, and `--group` flags on `role grant` and the `<group-id>` argument of the `group member` and `group role` commands take opaque IDs, not logins or names. Retrieve the ID first, then pass it. See [Scripting and Automation](youtrack-cli-scripting.md) for extracting a single field.
>
{style="warning"}

## Listing users

List all users with `youtrack user list`. Narrow the result with a query, or show the account your permanent token belongs to with `youtrack user current`:

```Shell
youtrack user list --query "banned: false"
youtrack user current
```

## Viewing a user

View one user by login:

```Shell
youtrack user view jane.doe
```

## Creating a user

Create an account with `youtrack user create`. The login, full name, email, and password are all required:

```Shell
youtrack user create --login jane.doe --fullName "Jane Doe" --email jane.doe@example.com --password "<password>"
```

YouTrack returns the new user, including its opaque `id`. By default the account is a `standard` user. Add `--type agent` or `--type reporter` to create the other account types.

> Pass the password through an environment variable or your shell's secret handling rather than a literal, so it does not land in shell history.
>
{style="warning"}

## Banning and unbanning a user

Ban an account to block sign-in while keeping its history:

```Shell
youtrack user ban jane.doe
youtrack user unban jane.doe
```

`user ban` asks for confirmation. Add `--force` to skip the prompt in scripts.

## Deleting a user

Deletion is permanent and requires a successor. YouTrack reassigns the deleted user's references — assignments, comments, and authored issues — to the successor:

```Shell
youtrack user delete jane.doe --successor john.doe
```

The command asks for confirmation. Add `--force` to skip the prompt in scripts. You can't delete the currently authenticated user, and the successor must be a different account.

> Prefer banning over deleting when you only need to revoke access. A banned account can be restored with `youtrack user unban`.
>
{style="tip"}

## Managing user preferences

View a user's general profile and set the locale and time zone:

```Shell
youtrack user profile view jane.doe
youtrack user locale set jane.doe en_US
youtrack user timezone set jane.doe Europe/Berlin
```

## Managing groups

Create, list, view, and delete groups. A group is created by name:

```Shell
youtrack group create --name "QA Team"
youtrack group list
youtrack group view "QA Team"
```

Delete a group and reassign its references to a successor group:

```Shell
youtrack group delete "QA Team" --successor Developers
```

`group delete` asks for confirmation. Add `--force` to skip the prompt in scripts.

### Listing group members

List the members of a group. This command takes the opaque group ID:

```Shell
youtrack group member list 2-9
```

### Assigning project roles to a group

Grant a role to every member of a group. Pass the opaque group ID as the argument and the opaque role ID through `--role`. Add `--project` to scope the role to one project; omit it for a global assignment:

```Shell
youtrack group role add 2-9 --role 2-3 --project 0-1
youtrack group role list 2-9
```

## Managing roles and permissions

A role bundles permissions. List existing roles and the permissions you can attach:

```Shell
youtrack role list
youtrack role permissions
```

View, create, update, and delete roles by opaque ID:

```Shell
youtrack role view 2-3
youtrack role create --name "Release Manager" --permissions "<permId1>,<permId2>"
youtrack role update 2-3 --permissions "<permId1>,<permId2>"
youtrack role delete 2-3
```

The `--permissions` flag takes a comma-separated list of permission IDs. Run `youtrack role permissions` to look them up. `role update` replaces the full permission set on the role.

Add or remove a single permission on an existing role without resetting the whole set:

```Shell
youtrack role permission list 2-3
youtrack role permission add 2-3 <permId>
youtrack role permission remove 2-3 <permId>
```

`role delete` and `role permission remove` ask for confirmation. Add `--force` to skip the prompt in scripts.

## Granting and revoking roles

Grant a role to a user or a group with `youtrack role grant`. Provide the role ID and exactly one of `--user` or `--group`, both opaque IDs. Add `--project` to scope the grant to a single project:

```Shell
youtrack role grant --role 2-3 --user 1-5
youtrack role grant --role 2-3 --group 2-9 --project 0-1
```

Revoking uses the assignment's own ID, not the user or role ID. Find it first with `role assignments`, then pass it to `role revoke`:

```Shell
youtrack role revoke <assignedRoleID>
```

`role revoke` asks for confirmation. Add `--force` to skip the prompt in scripts.

### Listing role assignments

List every role assignment, or filter to one holder with `--holder`. A holder is any principal a role can be granted to — a user, group, team, or organization — so `--holder` accepts an opaque ID of any of those types. The `--user` and `--group` flags are readability aliases for `--holder`; YouTrack does not distinguish them:

```Shell
youtrack role assignments
youtrack role assignments --holder 1-5
```

The `id` field of each row is the `<assignedRoleID>` you pass to `role revoke`.

## Managing organizations

List, create, and delete organizations. Delete takes the organization's opaque ID:

```Shell
youtrack org list
youtrack org create --name "Acme Inc"
youtrack org delete <id>
```

`org delete` asks for confirmation. Add `--force` to skip the prompt in scripts.

<seealso>
<category ref="user-guide">
<a href="youtrack-cli-managing-projects.md">Managing Projects</a>
<a href="youtrack-cli-administration.md">Administration</a>
<a href="youtrack-cli-scripting.md">Scripting and Automation</a>
</category>
<category ref="reference">
<a href="youtrack-cli.md">YouTrack CLI</a>
<a href="youtrack-cli-authentication.md">Authentication</a>
<a href="youtrack-cli-configuration.md">Configuration</a>
</category>
</seealso>
