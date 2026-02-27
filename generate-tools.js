#!/usr/bin/env node

// Generates AI tool definitions from the CLI's command registry.
//
// Usage:
//   node generate-tools.js           → Claude format  (uses input_schema)
//   node generate-tools.js --openai  → OpenAI format  (uses function.parameters)

import { execFileSync } from "child_process";
import { fileURLToPath } from "url";
import { dirname, join } from "path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const CLI = join(__dirname, "youtrack-cli.js");

// ── Parameter metadata ────────────────────────────────────────────────────────
// Provides a description and type for every parameter name used across all commands.
// When a model calls a tool, these descriptions are its only guidance for what value to supply.

const PARAM_META = {
  // Issues
  issueId:          { description: "Issue identifier, e.g. MYPROJ-42" },
  issueIds:         { description: "Comma-separated list of issue IDs, e.g. MYPROJ-1,MYPROJ-2" },
  linkedIssueId:    { description: "ID of the issue to link to" },
  newSummary:       { description: "Replacement summary text" },
  newDescription:   { description: "Replacement description text. Markdown supported." },
  idReadable:       { description: "Human-readable document identifier, e.g. MYPROJ-42" },

  // Common content
  summary:          { description: "One-line title" },
  description:      { description: "Full description. Markdown supported." },
  descriptionFile:  { description: "Path to a local file whose contents will be used as the description" },
  text:             { description: "Text content. Markdown supported." },
  textFile:         { description: "Path to a local file whose contents will be used as the text" },
  content:          { description: "Article body. Markdown supported." },
  contentFile:      { description: "Path to a local file whose contents will be used as the article content" },

  // Projects
  projectShortName: { description: "Project short name, e.g. MYPROJ" },
  projectId:        { description: "Project internal ID" },
  projectName:      { description: "Full project name (not the short name)" },
  shortName:        { description: "Short identifier for the project, e.g. MYPROJ" },
  projectIds:       { description: "Comma-separated list of project IDs to associate with the board" },
  leader:           { description: "Login of the user to set as project leader" },
  spentTimeField:   { description: "Name of the custom field used to track spent time" },
  estimationField:  { description: "Name of the custom field used for time estimation" },

  // Articles
  articleId:        { description: "Article identifier, e.g. MYPROJ-A-1" },
  parentArticleId:  { description: "ID of the parent article" },

  // Comments
  commentId:        { description: "Comment ID" },
  parentId:         { description: "Issue or article ID" },
  place:            { description: "'issues' (default) or 'articles'" },

  // Issue links
  linkType:         { description: "Link relationship type, e.g. 'depends on', 'duplicates', 'relates to'" },
  sourceToTarget:   { description: "Label for the source-to-target direction, e.g. 'depends on'" },
  targetToSource:   { description: "Label for the target-to-source direction, e.g. 'is required for'" },
  directed:         { description: "'true' if the link type is directional" },
  aggregation:      { description: "'true' if the link type is an aggregation" },

  // Custom fields
  fieldName:        { description: "Name of the custom field" },
  fieldValue:       { description: "Value to assign to the custom field" },
  fieldType:        { description: "Custom field type: enum, user, period, text, int, or string" },
  customFieldId:    { description: "Custom field ID" },
  fieldTypeId:      { description: "Custom field type ID, e.g. enum[1], user[1], date" },
  bundleType:       { description: "Bundle type, e.g. EnumBundle or StateBundle" },
  bundleId:         { description: "Bundle ID" },

  // Visibility
  permittedGroups:  { description: "Comma-separated group names to grant visibility to" },
  permittedUsers:   { description: "Comma-separated user logins to grant visibility to" },

  // Users
  userId:           { description: "User ID. Defaults to the current user if omitted." },
  userRingId:       { description: "User's Ring ID (Hub internal identifier)" },
  login:            { description: "User login name" },
  fullName:         { description: "User's full display name" },
  email:            { description: "User email address" },
  password:         { description: "User password" },

  // Groups & roles
  groupName:        { description: "Name of the user group" },
  groupId:          { description: "Hub group Ring ID" },
  roleName:         { description: "Role name" },
  roleId:           { description: "Role ID" },

  // User profile settings
  setting:          { description: "Setting key name" },
  value:            { description: "Setting value" },
  body:             { description: "Raw JSON as a string" },
  tips:             { description: "Comma-separated key=value pairs, e.g. showTips=true,showHelp=false" },
  key:              { description: "Property key" },
  locale:           { description: "Locale ID, e.g. en or de" },
  timezoneId:       { description: "Timezone ID, e.g. Europe/Berlin" },

  // Agile boards & sprints
  boardId:          { description: "Agile board ID" },
  template:         { description: "Board template: 'scrum' (default) or 'kanban'" },
  startDate:        { description: "Sprint start date as a Unix timestamp in milliseconds" },
  endDate:          { description: "Sprint end date as a Unix timestamp in milliseconds" },
  goal:             { description: "Sprint goal description" },

  // Work items
  minutes:          { description: "Duration in minutes" },
  workType:         { description: "Work item type name" },
  hoursInADay:      { description: "Number of working hours per day, e.g. 8" },
  autoAttach:       { description: "'true' to automatically attach this type to all projects" },

  // Workflows
  workflowId:       { description: "Workflow ID" },
  title:            { description: "Workflow display title" },
  script:           { description: "Workflow rule script content" },

  // Global settings
  feature:          { description: "Feature flag identifier" },
  enabled:          { description: "'true' or 'false'" },
  username:         { description: "License username" },
  license:          { description: "License key" },
  message:          { description: "Banner message text" },

  // Email settings
  serverAddress:    { description: "Mail server hostname" },
  port:             { description: "Port number" },
  fromAddress:      { description: "Sender email address" },
  replyToAddress:   { description: "Reply-to email address" },
  host:             { description: "Mail server hostname" },
  protocol:         { description: "Mail protocol: IMAP or POP3" },
  mailboxId:        { description: "Mailbox server ID" },
  disabled:         { description: "'true' to create the rule in a disabled state" },

  // Saved searches
  searchId:         { description: "Saved search ID" },
  users:            { description: "Comma-separated user logins" },
  groups:           { description: "Comma-separated group names" },

  // Tags
  tagId:            { description: "Tag ID" },

  // Dashboards
  dashboardId:      { description: "Dashboard ID" },

  // Reactions & attachments
  reaction:         { description: "Reaction emoji name, e.g. thumbsup" },
  reactionId:       { description: "Reaction ID" },
  attachmentId:     { description: "Attachment ID" },

  // Gantt & Canvas
  ganttId:          { description: "Gantt chart ID" },
  canvasId:         { description: "Planning canvas ID" },
  ownerLogin:       { description: "Login of the canvas owner" },
  label:            { description: "Card label text" },
  x:                { description: "Horizontal position on the canvas" },
  y:                { description: "Vertical position on the canvas" },
  nodeType:         { description: "Card node type, default 'CardNode'" },
  sourceCardId:     { description: "Source card ID" },
  targetCardId:     { description: "Target card ID" },

  // VCS & integrations
  serverId:         { description: "VCS server ID" },
  processorId:      { description: "VCS hosting processor ID" },

  // Organizations
  organizationId:   { description: "Organization ID" },

  // Apps & plugins
  appId:            { description: "App ID" },
  pluginId:         { description: "Plugin usage ID" },

  // Reports
  reportId:         { description: "Report ID" },

  // Auth
  tokenName:        { description: "Name for the generated token, default 'cli-token'" },
  scope:            { description: "Comma-separated service names to scope the token, e.g. YouTrack,Hub" },
  setEnv:           { description: "'true' to save the URL and token to ~/.youtrack-cli-env" },

  // Raw API
  path:             { description: "YouTrack API path, e.g. /api/issues?fields=id,summary" },

  // Generic
  name:             { description: "Display name" },
  query:            { description: "YouTrack search query, e.g. 'assignee: me State: Open'" },
  newName:          { description: "New name to apply" },
};

// ── Commands to exclude ───────────────────────────────────────────────────────
// list-commands is a meta command for introspection, not a YouTrack action.

const EXCLUDE = new Set(["list-commands"]);

// ── Build tool definitions ────────────────────────────────────────────────────

const format = process.argv.includes("--openai") ? "openai" : "claude";
const commands = JSON.parse(execFileSync("node", [CLI, "list-commands"], { encoding: "utf-8" }));

const tools = commands
  .filter(c => !EXCLUDE.has(c.name))
  .map(c => {
    const required = [];
    const properties = {};

    for (const p of c.params) {
      const isOptional = p.startsWith("?");
      const name = isOptional ? p.slice(1) : p;
      const meta = PARAM_META[name];
      properties[name] = {
        type: "string",
        description: meta ? meta.description : name,
      };
      if (!isOptional) required.push(name);
    }

    const schema = {
      type: "object",
      properties,
      ...(required.length > 0 ? { required } : {}),
    };

    if (format === "openai") {
      return {
        type: "function",
        function: {
          name: "youtrack_" + c.name.replace(/-/g, "_"),
          description: c.description,
          parameters: schema,
        },
      };
    }

    return {
      name: "youtrack_" + c.name.replace(/-/g, "_"),
      description: c.description,
      input_schema: schema,
    };
  });

console.log(JSON.stringify(tools, null, 2));
