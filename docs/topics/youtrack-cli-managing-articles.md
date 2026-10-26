[//]: # (title: Managing Articles)

<show-structure for="chapter" depth="2"/>

Articles are the documents in YouTrack's Knowledge Base. YouTrack CLI lets you create, read, update, and organize them from the command line, including comments, attachments, tags, visibility, and parent-child hierarchy. Each article belongs to a project and is written in Markdown.

Every article command takes an article ID, for example `DEMO-A-1`. To find one, run `youtrack article list` and copy the ID from the output.

## Listing articles

List articles with `youtrack article list`:

```Shell
youtrack article list
```

Narrow the results with a search query. The `--query` flag accepts the same syntax as the Knowledge Base search box:

```Shell
youtrack article list --query "project: DEMO"
```

## Viewing an article

View a single article by its ID with `youtrack article view`:

```Shell
youtrack article view DEMO-A-1
```

The output includes the article's summary and Markdown content. Add `--json` for the full, machine-readable record. See [Scripting and Automation](youtrack-cli-scripting.md) for output formats.

## Creating an article

Create an article with `youtrack article create`. Specify the project by its short name and provide a title with `--summary`:

```Shell
youtrack article create --project DEMO --summary "Deployment checklist"
```

YouTrack returns the new article, including its ID. To add body text, pass Markdown to `--content`. There is no flag for reading content from a file, so expand a Markdown file into `--content` with your shell:

```Shell
youtrack article create --project DEMO --summary "Runbook" \
  --content "$(cat runbook.md)"
```

### article create flags

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

`--project`
</td>
<td>

Short name of the project that owns the article. Required.
</td>
</tr>
<tr>
<td>

`--summary`
</td>
<td>

Article title. Required.
</td>
</tr>
<tr>
<td>

`--content`
</td>
<td>

Article body, written in Markdown.
</td>
</tr>
<tr>
<td>

`--asUser`
</td>
<td>

Login of the user to record as the author. The target user must have at least a Contributor role in the project.
</td>
</tr>
</table>

> Use `youtrack project list` to look up a project's short name.
>
{style="tip"}

## Updating an article

Change an article's title or content with `youtrack article update`. Pass only the fields you want to change:

```Shell
youtrack article update DEMO-A-1 --summary "Deployment checklist (v2)"
```

The `--content` flag replaces the entire body. It does not append to the existing content.

## Deleting an article

Delete an article with `youtrack article delete`. The CLI asks for confirmation first; add `--force` to skip the prompt in a script:

```Shell
youtrack article delete DEMO-A-1
youtrack article delete DEMO-A-1 --force
```

> Deleting an article is permanent. Deleting a parent article also affects its child articles.
>
{style="warning"}

## Organizing articles in a hierarchy

Articles form a tree. Every article can have one parent and any number of children.

Set an article's parent with `youtrack article set-parent`. The first argument is the article, the second is the new parent:

```Shell
youtrack article set-parent DEMO-A-3 DEMO-A-1
```

Read the hierarchy with `youtrack article parent view` and `youtrack article child list`:

```Shell
youtrack article parent view DEMO-A-3
youtrack article child list DEMO-A-1
```

You can also attach a child from the parent's side with `youtrack article child add <parentId> <childId>`.

## Commenting on articles

List the comments on an article with `youtrack article comment list`, view a single comment by its ID with `youtrack article comment view`, and add one with `youtrack article comment create` (the text is required):

```Shell
youtrack article comment list DEMO-A-1
youtrack article comment view DEMO-A-1 2-45
youtrack article comment create DEMO-A-1 --text "Reviewed and approved."
```

Update a comment by passing the article ID and the comment ID:

```Shell
youtrack article comment update DEMO-A-1 2-45 --text "Approved after edits."
```

Delete a comment the same way. Add `--force` to skip the confirmation prompt:

```Shell
youtrack article comment delete DEMO-A-1 2-45 --force
```

Pin a comment to the top of the article, or unpin it, with `youtrack article comment pin`:

```Shell
youtrack article comment pin DEMO-A-1 2-45
youtrack article comment pin DEMO-A-1 2-45 --unpin
```

Restrict who can see a comment with `youtrack article comment visibility`. Both `--groups` and `--users` take comma-separated opaque IDs:

```Shell
youtrack article comment visibility DEMO-A-1 2-45 --groups 3-2 --users 2-9
```

### Reacting to a comment

List the reactions on a comment with `youtrack article comment reaction list`, and add one with `youtrack article comment reaction add`. Remove a reaction with `youtrack article comment reaction delete`, which prompts for confirmation unless you pass `--force`:

```Shell
youtrack article comment reaction add DEMO-A-1 2-45 --reaction "+1"
youtrack article comment reaction delete DEMO-A-1 2-45 4-12 --force   # 4-12 is the reaction ID from `reaction list`
```

## Attaching files

List an article's attachments with `youtrack article attachment list`, and attach a file with `youtrack article attachment add`. Point `--file` at a local path:

```Shell
youtrack article attachment list DEMO-A-1
youtrack article attachment add DEMO-A-1 --file ./architecture.png
youtrack article attachment delete DEMO-A-1 5-12
```

## Tagging an article

Attach an existing tag to an article with `youtrack article tag`. The second argument is the tag's ID, not its name:

```Shell
youtrack article tag DEMO-A-1 6-12
youtrack article untag DEMO-A-1 6-12
```

> Look up a tag's ID with `youtrack tag list --query <name>` before tagging.
>
{style="tip"}

## Setting article visibility

By default an article is visible to everyone who can see its project. Restrict it to specific groups or users with `youtrack article visibility`. Both flags take comma-separated opaque IDs:

```Shell
youtrack article visibility DEMO-A-1 --groups 3-2,3-7 --users 2-9
```

Retrieve the IDs first: run `youtrack group list` for group IDs and `youtrack user view jane.doe` for a user ID. See [Managing Users and Access](youtrack-cli-managing-users-and-access.md) for details.

<seealso>
<category ref="user-guide">
<a href="youtrack-cli-managing-issues.md">Managing Issues</a>
<a href="youtrack-cli-managing-projects.md">Managing Projects</a>
<a href="youtrack-cli-managing-users-and-access.md">Managing Users and Access</a>
<a href="youtrack-cli-scripting.md">Scripting and Automation</a>
</category>
<category ref="reference">
<a href="youtrack-cli-configuration.md">Configuration</a>
<a href="youtrack-cli-authentication.md">Authentication</a>
</category>
</seealso>
