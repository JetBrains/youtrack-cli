# Managing Articles

<show-structure for="chapter" depth="2"/>

Articles are the Markdown documents in the YouTrack Knowledge Base, and each article belongs to a project. This page shows how to create, read, update, and organize articles with the `youtrack article` commands, including comments, attachments, tags, visibility, and the parent-child hierarchy.

Most article commands take an article ID, for example `DEMO-A-1`. To find an ID, run `youtrack article list`.

## List and search articles

List articles with `youtrack article list`:

```Shell
youtrack article list
```

The command returns up to 100 articles by default. Use `--limit` to change the number. `--limit 0` returns no articles. Narrow the results with a search query. The `--query` flag accepts the same syntax as the Knowledge Base search box:

```Shell
youtrack article list --query "project: DEMO" --limit 20
```

## View one article by its ID

View a single article by its ID with `youtrack article view`:

```Shell
youtrack article view DEMO-A-1
```

The output includes the summary and the Markdown content of the article. Add `--json` for the full, machine-readable record. See [Scripting and Automation](youtrack-cli-scripting.md) for output formats.

## Create an article in a project

Create an article with `youtrack article create`. Specify the project by its short name and give a title with `--summary`:

```Shell
youtrack article create --project DEMO --summary "Deployment checklist"
```

> Use `youtrack project list` to look up a project's short name.

YouTrack returns the new article, including its ID. To add body text, pass Markdown to `--content`. No flag reads content from a file. To use a Markdown file, expand it into `--content` with your shell:

```Shell
youtrack article create --project DEMO --summary "Runbook" \
  --content "$(cat runbook.md)"
```

### Flags of article create

| Flag | Description |
|---|---|
| `--project` | Short name of the project that owns the article. Required. |
| `--summary` | Article title. Required. |
| `--content` | Article body, in Markdown. |
| `--asUser` | Login of the user to record as the author. The target user must have at least a Contributor role in the project. |

## Update replaces the fields you pass

Change the title or the content of an article with `youtrack article update`. Pass only the fields you want to change:

```Shell
youtrack article update DEMO-A-1 --summary "Deployment checklist (v2)"
```

The `--content` flag replaces the full body. It does not append to the content.

## Delete an article permanently

<warning>

Deleting an article is permanent. The CLI does not check whether the article has child articles. Before you delete a parent article, run `youtrack article child list <articleId>` to see its children.

</warning>

Delete an article with `youtrack article delete`. The CLI asks for confirmation. Add `--force` to skip the prompt in a script:

```Shell
youtrack article delete DEMO-A-1
youtrack article delete DEMO-A-1 --force
```

## Organize articles in a tree

Articles form a tree. Every article can have one parent and any number of children.

Set the parent of an article with `youtrack article set-parent`. The first argument is the article. The second argument is the new parent:

```Shell
youtrack article set-parent DEMO-A-3 DEMO-A-1
```

Read the hierarchy with `youtrack article parent view` and `youtrack article child list`:

```Shell
youtrack article parent view DEMO-A-3
youtrack article child list DEMO-A-1
```

To set the relation from the side of the parent, use `youtrack article child add <parentId> <childId>`.

## Comment on articles

List the comments on an article with `youtrack article comment list`. View a single comment by its ID with `youtrack article comment view`. Add a comment with `youtrack article comment create`. The `--text` flag is required:

```Shell
youtrack article comment list DEMO-A-1
youtrack article comment view DEMO-A-1 2-45
youtrack article comment create DEMO-A-1 --text "Reviewed and approved."
```

Update a comment with the article ID and the comment ID:

```Shell
youtrack article comment update DEMO-A-1 2-45 --text "Approved after edits."
```

Delete a comment in the same way. The command asks for confirmation. Add `--force` to skip the prompt:

```Shell
youtrack article comment delete DEMO-A-1 2-45 --force
```

Pin a comment to the top of the article, or unpin it, with `youtrack article comment pin`:

```Shell
youtrack article comment pin DEMO-A-1 2-45
youtrack article comment pin DEMO-A-1 2-45 --unpin
```

### Restrict who can see a comment

<warning>

`article comment visibility` without `--groups` and without `--users` sends empty lists. YouTrack treats empty lists as no restriction, so the comment becomes visible to everyone who can see the article.

</warning>

Restrict who can see a comment with `youtrack article comment visibility`. `--groups` and `--users` take comma-separated opaque IDs. Each call replaces both lists: a list you leave out is cleared.

```Shell
youtrack article comment visibility DEMO-A-1 2-45 --groups 3-2 --users 1-5
```

### React to a comment

List the reactions on a comment with `youtrack article comment reaction list`. Add a reaction with `youtrack article comment reaction add` and the `--reaction` flag. Remove a reaction by its reaction ID with `youtrack article comment reaction delete`. The delete command asks for confirmation unless you pass `--force`:

```Shell
youtrack article comment reaction list DEMO-A-1 2-45
youtrack article comment reaction add DEMO-A-1 2-45 --reaction "👍"
youtrack article comment reaction delete DEMO-A-1 2-45 4-12 --force   # 4-12 is the reaction ID from `reaction list`
```

Issue comments use a different syntax: they take the reaction as a positional argument (see [Managing Issues](youtrack-cli-managing-issues.md#react-to-a-comment)).

## Attach files to an article

List the attachments of an article with `youtrack article attachment list`. Attach a file with `youtrack article attachment add`, and point `--file` at a local path. Delete an attachment by its ID with `youtrack article attachment delete`. The delete command asks for confirmation unless you pass `--force`:

```Shell
youtrack article attachment list DEMO-A-1
youtrack article attachment add DEMO-A-1 --file ./architecture.png
youtrack article attachment delete DEMO-A-1 5-12
```

## Tag an article by tag ID

> Look up the ID of a tag with `youtrack tag list --query <name>` before you tag an article.

Attach an existing tag to an article with `youtrack article tag`, and remove it with `youtrack article untag`. The second argument is the tag ID, not the tag name:

```Shell
youtrack article tag DEMO-A-1 6-12
youtrack article untag DEMO-A-1 6-12
```

## Restrict article visibility to groups and users

By default, everyone who can see the project can see the article. To restrict an article to specific groups or users, use `youtrack article visibility`.

<warning>

Each call replaces both lists: a list you leave out is cleared. If you pass neither `--groups` nor `--users`, the command sends empty lists. YouTrack treats empty lists as no restriction, so the article becomes visible to everyone who can see the project. Use this call to remove a restriction.

</warning>

Both flags take comma-separated opaque IDs:

```Shell
youtrack article visibility DEMO-A-1 --groups 3-2,3-7 --users 1-5
```

Get the IDs first: run `youtrack group list` for group IDs and `youtrack user view jane.doe` for a user ID. See [Managing Users and Access](youtrack-cli-managing-users-and-access.md) for details.

<seealso>
<category ref="04">
<a href="youtrack-cli-managing-issues.md">Managing Issues</a>
<a href="youtrack-cli-managing-projects.md">Managing Projects</a>
<a href="youtrack-cli-managing-users-and-access.md">Managing Users and Access</a>
<a href="youtrack-cli-scripting.md">Scripting and Automation</a>
</category>
<category ref="06">
<a href="youtrack-cli-configuration.md">Configuration</a>
<a href="youtrack-cli-authentication.md">Authentication</a>
</category>
</seealso>
