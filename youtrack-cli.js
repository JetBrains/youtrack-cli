#!/usr/bin/env node

// YouTrack REST API CLI — an internal tool for AI agents and developers to interact with YouTrack.
// Usage: node youtrack-cli.js <command> [options]
// Env:   YOUTRACK_BASE_URL, YOUTRACK_TOKEN

import { writeFileSync, existsSync, readFileSync } from "fs";
import { homedir } from "os";
import { join } from "path";

import { API } from "./api-paths.js";
import { basename, extname } from "path";
import {
  doGet, doPost, doPut, doDelete, doMultipart, httpReq, httpReqBasicAuth,
  enc, resolveContent, parseVisibility, parseBool, parseUserIdOrMe, parseIds,
  findByName, findByLogin, findByShortName, setVerbose,
} from "./http-helpers.js";

const MIME_TYPES = {
  ".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
  ".gif": "image/gif", ".webp": "image/webp", ".svg": "image/svg+xml",
  ".pdf": "application/pdf", ".txt": "text/plain", ".md": "text/markdown",
  ".json": "application/json", ".zip": "application/zip",
};

const USER_TYPES = ["STANDARD_USER", "AGENT", "REPORTER"];

// ── Helpers ───────────────────────────────────────────────────────────────────

// YouTrack's command parser requires braces around logins that contain dots or
// hyphens (e.g. "assignee alice.qa" must become "assignee {alice.qa}").
function wrapUserValues(query) {
  return query.replace(
    /\b(assignee|reporter)\s+(?!\{)(\S*[.\-@]\S+)/gi,
    (_, field, value) => `${field} {${value}}`
  );
}

// Look up a user by login and return { id }.
async function resolveUserRef(base, token, login) {
  const users = await doGet(base, `${API.USERS}?query=${enc(login)}&fields=id,login`, token);
  return findByLogin(users, login, "User");
}

// Poll until a user with the given login appears in YouTrack (Hub sync complete).

// Filter a result object/array to only the specified dot-path fields.
function filterFields(data, spec) {
  const tree = {};
  for (const path of spec) {
    const parts = path.split(".");
    let node = tree;
    for (let i = 0; i < parts.length - 1; i++) { node = node[parts[i]] = node[parts[i]] || {}; }
    node[parts[parts.length - 1]] = true;
  }
  function apply(obj, t) {
    if (Array.isArray(obj)) return obj.map(el => apply(el, t));
    if (!obj || typeof obj !== "object") return obj;
    const out = {};
    for (const [k, v] of Object.entries(t)) if (k in obj) out[k] = v === true ? obj[k] : apply(obj[k], v);
    return out;
  }
  return apply(data, tree);
}

// Parse flexible date input: ms timestamp, relative ("7d", "-2d", "24h"), or ISO date.
function parseDate(value) {
  if (!value) return undefined;
  if (/^\d+$/.test(value)) return parseInt(value);
  const rel = value.match(/^(-?\d+)(d|h)$/);
  if (rel) return Date.now() + parseInt(rel[1]) * (rel[2] === "d" ? 86400000 : 3600000);
  const d = new Date(value);
  if (!isNaN(d.getTime())) return d.getTime();
  throw new Error(`Invalid date: "${value}". Use ms timestamp, relative ("7d", "-2d", "24h"), or ISO date ("YYYY-MM-DD").`);
}

// Heuristic: commands whose names start with write prefixes are write operations.
const _WRITE_PREFIXES = ["add-", "delete-", "update-", "set-", "apply-", "link-", "unlink-",
  "ban-", "unban-", "grant-", "revoke-", "archive-", "convert-", "attach-", "pin-",
  "create-", "restore-", "disable-", "enable-", "switch-", "remove-", "tag-"];
const _EXTRA_WRITE = new Set(["get-token", "raw-post", "raw-delete"]);
function isWriteCommand(name) {
  return _EXTRA_WRITE.has(name) || _WRITE_PREFIXES.some(p => name.startsWith(p));
}

const COMMAND_SUGGESTIONS = {
  "add-project": "Ensure --leader is a valid user login. Run: youtrack-cli get-current-user",
  "set-custom-field": "Run get-custom-field-prototypes to verify fieldName. fieldType must be one of: enum, user, period, text, int, string",
  "add-bundle-value": "Run get-bundles to verify bundle name and type",
  "add-issue": "Ensure --projectShortName is a valid project short name. Run: youtrack-cli get-projects",
  "apply-command": "Check command syntax. Run: youtrack-cli get-command-assist --query <query> --issueIds <id>",
};

// ── Command registry ──────────────────────────────────────────────────────────

const COMMANDS = {};

function cmd(name, description, params, handler) {
  COMMANDS[name] = { description, params, handler };
}

// ── Issues ────────────────────────────────────────────────────────────────────

cmd("add-issue", "Create an issue", ["projectShortName", "summary", "?description", "?descriptionFile", "?asUser"], async (ctx, args) => {
  const description = resolveContent(args.description, args.descriptionFile);
  const projects = await doGet(ctx.base, `${API.PROJECTS}?query=${enc(args.projectShortName)}&fields=id,shortName&$top=1`, ctx.token);
  const proj = findByShortName(projects, args.projectShortName, "Project");
  const body = { summary: args.summary, description, project: { id: proj.id } };
  if (args.asUser) {
    const u = await resolveUserRef(ctx.base, ctx.token, args.asUser);
    body.reporter = { id: u.id };
  }
  return doPost(ctx.base, `${API.ISSUES}?fields=id,idReadable,summary,description,project,reporter(id,login)`, ctx.token, body);
});

cmd("get-issue", "Get issue by ID", ["issueId"], async (ctx, args) => {
  return doGet(ctx.base, `${API.ISSUES}/${enc(args.issueId)}?fields=id,idReadable,summary,description,resolved,created`, ctx.token);
});

cmd("get-issues", "Search issues by query", ["query", "?limit"], async (ctx, args) => {
  const top = args.limit ? parseInt(args.limit, 10) : 100;
  return doGet(ctx.base, `${API.ISSUES}?query=${enc(args.query)}&$top=${top}&fields=id,summary,description,idReadable,project`, ctx.token);
});

cmd("update-issue-summary", "Update issue summary", ["issueId", "newSummary"], async (ctx, args) => {
  return doPost(ctx.base, `${API.ISSUES}/${enc(args.issueId)}?fields=summary`, ctx.token, { summary: args.newSummary });
});

cmd("update-issue-description", "Update issue description", ["issueId", "newDescription"], async (ctx, args) => {
  return doPost(ctx.base, `${API.ISSUES}/${enc(args.issueId)}?fields=description`, ctx.token, { description: args.newDescription });
});

cmd("delete-issue", "Delete issue", ["issueId"], async (ctx, args) => {
  return doDelete(ctx.base, `${API.ISSUES}/${enc(args.issueId)}`, ctx.token);
});

cmd("apply-command", "Apply command to issues (e.g. 'State Fixed')", ["query", "issueIds"], async (ctx, args) => {
  const issues = await Promise.all(
    args.issueIds.split(",").map(async id => {
      const issue = await doGet(ctx.base, `${API.ISSUES}/${enc(id.trim())}?fields=id`, ctx.token);
      return { id: issue.id };
    })
  );
  return doPost(ctx.base, API.COMMANDS, ctx.token, { query: wrapUserValues(args.query), issues });
});

cmd("get-command-assist", "Get command suggestions/validation", ["query", "issueIds"], async (ctx, args) => {
  const issues = await Promise.all(
    args.issueIds.split(",").map(async id => {
      const issue = await doGet(ctx.base, `${API.ISSUES}/${enc(id.trim())}?fields=id`, ctx.token);
      return { id: issue.id };
    })
  );
  return doPost(ctx.base, `${API.COMMANDS}/assist?fields=commands(error,description),query`, ctx.token, { query: args.query, issues });
});

// ── Issue Visibility ──────────────────────────────────────────────────────────

cmd("update-issue-visibility", "Update issue visibility", ["issueId", "?permittedGroups", "?permittedUsers"], async (ctx, args) => {
  const allGroups = await doGet(ctx.base, `${API.GROUPS}?fields=id,name&$top=-1`, ctx.token);
  const body = parseVisibility(args, allGroups);
  return doPost(ctx.base, `${API.ISSUES}/${enc(args.issueId)}?fields=id,visibility`, ctx.token, body);
});

// ── Issue Custom Fields ───────────────────────────────────────────────────────

cmd("get-custom-field-value", "Get custom field value", ["issueId", "fieldName"], async (ctx, args) => {
  const fields = await doGet(ctx.base, `${API.ISSUES}/${enc(args.issueId)}/fields?fields=id,name,value(name,presentation,text,login,minutes)`, ctx.token);
  return findByName(fields, args.fieldName, "Field");
});

cmd("set-custom-field", "Set single-value custom field", ["issueId", "fieldName", "fieldValue", "fieldType"], async (ctx, args) => {
  const fields = await doGet(ctx.base, `${API.ISSUES}/${enc(args.issueId)}/fields?fields=id,name`, ctx.token);
  const field = findByName(fields, args.fieldName, "Field");
  const type = args.fieldType;
  let body;
  switch (type) {
    case "period": body = { value: { presentation: args.fieldValue } }; break;
    case "user": {
      const users = await doGet(ctx.base, `${API.USERS}?query=${enc(args.fieldValue)}&fields=id,login`, ctx.token);
      const u = findByLogin(users, args.fieldValue, "User");
      body = { value: { id: u.id } }; break;
    }
    case "text": body = { value: { text: args.fieldValue } }; break;
    case "int": body = { value: parseInt(args.fieldValue) }; break;
    case "string": body = { value: args.fieldValue }; break;
    default: body = { value: { name: args.fieldValue } };
  }
  return doPost(ctx.base, `${API.ISSUES}/${enc(args.issueId)}/fields/${field.id}`, ctx.token, body);
});

// ── Comments (issues and articles) ───────────────────────────────────────────
// All comment commands accept ?place ("issues" | "articles", default "issues").

cmd("add-comment", "Add comment to issue or article", ["parentId", "?text", "?textFile", "?place", "?asUser"], async (ctx, args) => {
  const place = args.place || "issues";
  const basePath = place === "articles" ? API.ARTICLES : API.ISSUES;
  const text = resolveContent(args.text, args.textFile);
  if (!text) throw new Error("Either --text or --textFile is required");
  const body = { text };
  if (args.asUser) {
    const u = await resolveUserRef(ctx.base, ctx.token, args.asUser);
    body.author = { id: u.id };
  }
  return doPost(ctx.base, `${basePath}/${enc(args.parentId)}/comments?fields=id,text,author(id,login)`, ctx.token, body);
});

cmd("get-comments", "Get comments for issue or article", ["parentId", "?place"], async (ctx, args) => {
  const place = args.place || "issues";
  const basePath = place === "articles" ? API.ARTICLES : API.ISSUES;
  return doGet(ctx.base, `${basePath}/${enc(args.parentId)}/comments?$top=100&fields=id,text,author,created`, ctx.token);
});

cmd("update-comment", "Edit a comment", ["parentId", "commentId", "?text", "?textFile", "?place"], async (ctx, args) => {
  const place = args.place || "issues";
  const basePath = place === "articles" ? API.ARTICLES : API.ISSUES;
  const text = resolveContent(args.text, args.textFile);
  if (!text) throw new Error("Either --text or --textFile is required");
  return doPost(ctx.base, `${basePath}/${enc(args.parentId)}/comments/${enc(args.commentId)}?fields=id,text`, ctx.token, { text });
});

cmd("delete-comment", "Soft-delete a comment", ["parentId", "commentId", "?place"], async (ctx, args) => {
  const place = args.place || "issues";
  const basePath = place === "articles" ? API.ARTICLES : API.ISSUES;
  return doPost(ctx.base, `${basePath}/${enc(args.parentId)}/comments/${enc(args.commentId)}?fields=id,text`, ctx.token, { deleted: true });
});

cmd("update-comment-visibility", "Update comment visibility", ["parentId", "commentId", "?place", "?permittedGroups", "?permittedUsers"], async (ctx, args) => {
  const place = args.place || "issues";
  const basePath = place === "articles" ? API.ARTICLES : API.ISSUES;
  const allGroups = await doGet(ctx.base, `${API.GROUPS}?fields=id,name&$top=-1`, ctx.token);
  const body = parseVisibility(args, allGroups);
  return doPost(ctx.base, `${basePath}/${enc(args.parentId)}/comments/${enc(args.commentId)}?fields=id,visibility`, ctx.token, body);
});

cmd("pin-comment", "Pin or unpin a comment", ["parentId", "commentId", "?place", "?pinned"], async (ctx, args) => {
  const place = args.place || "issues";
  const basePath = place === "articles" ? API.ARTICLES : API.ISSUES;
  const pinned = args.pinned !== undefined ? parseBool(args.pinned) : true;
  return doPost(ctx.base, `${basePath}/${enc(args.parentId)}/comments/${enc(args.commentId)}?fields=id,pinned`, ctx.token, { pinned });
});

// ── Issue Links ───────────────────────────────────────────────────────────────

cmd("link-issues", "Link two issues", ["issueId", "linkedIssueId", "linkType"], async (ctx, args) => {
  const linkTypes = await doGet(ctx.base, `${API.LINK_TYPES}?fields=id,sourceToTarget,targetToSource`, ctx.token);
  const inward = ["depends on", "duplicates", "subtask of"];
  let linkId, suffix;
  if (inward.includes(args.linkType)) {
    const lt = findByName(linkTypes.map(l => ({ ...l, name: l.targetToSource })), args.linkType, "Link type");
    linkId = lt.id; suffix = "t";
  } else {
    const lt = findByName(linkTypes.map(l => ({ ...l, name: l.sourceToTarget })), args.linkType, "Link type");
    linkId = lt.id; suffix = "s";
  }
  const linked = await doGet(ctx.base, `${API.ISSUES}/${enc(args.linkedIssueId)}?fields=id`, ctx.token);
  return doPost(ctx.base, `${API.ISSUES}/${enc(args.issueId)}/links/${linkId}${suffix}/issues`, ctx.token, { id: linked.id });
});

cmd("add-issue-link-type", "Create issue link type", ["name", "sourceToTarget", "?targetToSource", "?directed", "?aggregation"], async (ctx, args) => {
  return doPost(ctx.base, `${API.LINK_TYPES}?fields=id,name,sourceToTarget,targetToSource,aggregation,directed`, ctx.token, {
    name: args.name,
    sourceToTarget: args.sourceToTarget,
    targetToSource: args.targetToSource || "",
    directed: args.directed === "true",
    aggregation: args.aggregation === "true",
  });
});

cmd("get-issue-link-types", "List all issue link types", [], async (ctx) => {
  return doGet(ctx.base, `${API.LINK_TYPES}?fields=id,name,sourceToTarget,targetToSource,aggregation,directed`, ctx.token);
});

cmd("get-linked-issues", "Get issues linked via a specific link type", ["issueId", "linkType"], async (ctx, args) => {
  const linkTypes = await doGet(ctx.base, `${API.LINK_TYPES}?fields=id,sourceToTarget,targetToSource`, ctx.token);
  const inward = ["depends on", "duplicates", "subtask of"];
  let linkId, suffix;
  if (inward.includes(args.linkType)) {
    const lt = findByName(linkTypes.map(l => ({ ...l, name: l.targetToSource })), args.linkType, "Link type");
    linkId = lt.id; suffix = "t";
  } else {
    const lt = findByName(linkTypes.map(l => ({ ...l, name: l.sourceToTarget })), args.linkType, "Link type");
    linkId = lt.id; suffix = "s";
  }
  return doGet(ctx.base, `${API.ISSUES}/${enc(args.issueId)}/links/${linkId}${suffix}/issues?fields=id,idReadable,summary`, ctx.token);
});

cmd("remove-issue-link", "Remove a link between two issues", ["issueId", "linkedIssueId", "linkType"], async (ctx, args) => {
  const linkTypes = await doGet(ctx.base, `${API.LINK_TYPES}?fields=id,sourceToTarget,targetToSource`, ctx.token);
  const inward = ["depends on", "duplicates", "subtask of"];
  let linkId, suffix;
  if (inward.includes(args.linkType)) {
    const lt = findByName(linkTypes.map(l => ({ ...l, name: l.targetToSource })), args.linkType, "Link type");
    linkId = lt.id; suffix = "t";
  } else {
    const lt = findByName(linkTypes.map(l => ({ ...l, name: l.sourceToTarget })), args.linkType, "Link type");
    linkId = lt.id; suffix = "s";
  }
  const linked = await doGet(ctx.base, `${API.ISSUES}/${enc(args.linkedIssueId)}?fields=id`, ctx.token);
  return doDelete(ctx.base, `${API.ISSUES}/${enc(args.issueId)}/links/${linkId}${suffix}/issues/${linked.id}`, ctx.token);
});

cmd("get-issue-count", "Count issues matching a query", ["query"], async (ctx, args) => {
  return doPost(ctx.base, `${API.ISSUE_COUNT}?fields=count`, ctx.token, { query: args.query });
});

// ── Articles ──────────────────────────────────────────────────────────────────

cmd("add-article", "Create article", ["projectShortName", "summary", "?content", "?contentFile", "?asUser"], async (ctx, args) => {
  const content = resolveContent(args.content, args.contentFile);
  const projects = await doGet(ctx.base, `${API.PROJECTS}?query=${enc(args.projectShortName)}&fields=id,shortName&$top=1`, ctx.token);
  const proj = findByShortName(projects, args.projectShortName, "Project");
  const draft = await doPost(ctx.base, `${API.USERS_ME}/articleDrafts?fields=id`, ctx.token, {
    project: { id: proj.id },
    summary: args.summary,
    content,
  });
  let article = await doPost(ctx.base, `${API.ARTICLES}?draftId=${draft.id}&fields=id,idReadable,summary,content`, ctx.token, {});
  if (args.asUser) {
    const u = await resolveUserRef(ctx.base, ctx.token, args.asUser);
    article = await doPost(ctx.base, `${API.ARTICLES}/${article.id}?fields=id,idReadable,summary,content,reporter(id,login)`, ctx.token, { reporter: { id: u.id } });
  }
  return article;
});

cmd("get-articles", "List or search articles", ["?query"], async (ctx, args) => {
  const q = args.query ? `?query=${enc(args.query)}&` : "?";
  return doGet(ctx.base, `${API.ARTICLES}${q}$top=100&fields=id,summary,idReadable,parentArticle(idReadable)`, ctx.token);
});

cmd("get-article", "Get article by ID", ["articleId"], async (ctx, args) => {
  return doGet(ctx.base, `${API.ARTICLES}/${enc(args.articleId)}?fields=id,summary,content,idReadable`, ctx.token);
});

cmd("delete-article", "Delete article", ["articleId"], async (ctx, args) => {
  return doDelete(ctx.base, `${API.ARTICLES}/${enc(args.articleId)}`, ctx.token);
});

cmd("update-article-summary", "Update article summary", ["articleId", "newSummary"], async (ctx, args) => {
  return doPost(ctx.base, `${API.ARTICLES}/${enc(args.articleId)}?fields=id,summary`, ctx.token, { summary: args.newSummary });
});

cmd("update-article-content", "Update article content", ["articleId", "?newContent", "?newContentFile"], async (ctx, args) => {
  const content = resolveContent(args.newContent, args.newContentFile);
  return doPost(ctx.base, `${API.ARTICLES}/${enc(args.articleId)}?fields=id,content`, ctx.token, { content });
});

cmd("set-parent-article", "Set parent article", ["articleId", "parentArticleId"], async (ctx, args) => {
  return doPost(ctx.base, `${API.ARTICLES}/${enc(args.articleId)}?fields=id,parentArticle(idReadable)`, ctx.token, { parentArticle: { idReadable: args.parentArticleId } });
});

cmd("update-article-visibility", "Update article visibility", ["articleId", "?permittedGroups", "?permittedUsers"], async (ctx, args) => {
  const allGroups = await doGet(ctx.base, `${API.GROUPS}?fields=id,name&$top=-1`, ctx.token);
  const body = parseVisibility(args, allGroups);
  return doPost(ctx.base, `${API.ARTICLES}/${enc(args.articleId)}?fields=id,visibility`, ctx.token, body);
});

cmd("tag-article", "Tag an article", ["articleId", "tagId"], async (ctx, args) => {
  return doPost(ctx.base, `${API.ARTICLES}/${enc(args.articleId)}/tags?fields=id`, ctx.token, { $type: "Tag", id: args.tagId });
});

cmd("get-removed-documents", "List removed (trashed) documents", [], async (ctx) => {
  return doGet(ctx.base, `${API.REMOVED_DOCUMENTS}?fields=id,idReadable,summary,deletedTime,$type,project(id)&query=&$top=100`, ctx.token);
});

cmd("restore-document", "Restore a removed document", ["idReadable"], async (ctx, args) => {
  const docs = await doGet(ctx.base, `${API.REMOVED_DOCUMENTS}?fields=id,idReadable,$type&query=&$top=100`, ctx.token);
  const doc = (Array.isArray(docs) ? docs : []).find(d => d.idReadable === args.idReadable);
  if (!doc) throw new Error(`Removed document '${args.idReadable}' not found`);
  return doPost(ctx.base, `${API.REMOVED_DOCUMENTS}?fields=id,idReadable`, ctx.token, {
    documents: [{ id: doc.id, $type: doc.$type }],
    remove: false,
  });
});

// ── Projects ──────────────────────────────────────────────────────────────────

cmd("add-project", "Create project", ["shortName", "name", "?leader"], async (ctx, args) => {
  const body = { shortName: args.shortName, name: args.name };
  if (args.leader) {
    const users = await doGet(ctx.base, `${API.USERS}?query=${enc(args.leader)}&fields=id,login`, ctx.token);
    const u = findByLogin(users, args.leader, "User");
    body.leader = { id: u.id };
  }
  return doPost(ctx.base, `${API.PROJECTS}?fields=id,shortName,name`, ctx.token, body);
});

cmd("get-projects", "List all projects", [], async (ctx) => {
  return doGet(ctx.base, `${API.PROJECTS}?fields=id,shortName,name,archived&$top=-1`, ctx.token);
});

cmd("get-project", "Get project by short name", ["shortName"], async (ctx, args) => {
  const projects = await doGet(ctx.base, `${API.PROJECTS}?query=${enc(args.shortName)}&fields=id,shortName,name&$top=10`, ctx.token);
  return projects.find(p => p.shortName === args.shortName) || null;
});

cmd("delete-project", "Delete project", ["projectId"], async (ctx, args) => {
  return doDelete(ctx.base, `${API.PROJECTS}/${enc(args.projectId)}`, ctx.token);
});

cmd("archive-project", "Archive project", ["projectId"], async (ctx, args) => {
  return doPost(ctx.base, `${API.PROJECTS}/${enc(args.projectId)}`, ctx.token, { archived: true });
});

cmd("get-project-templates", "List project templates", [], async (ctx) => {
  return doGet(ctx.base, `/api/admin/projectTemplates?fields=id,name,shortName&$top=-1`, ctx.token);
});

cmd("convert-project-to-template", "Convert project to template", ["projectId"], async (ctx, args) => {
  return doPost(ctx.base, `${API.PROJECTS}/${enc(args.projectId)}?fields=id`, ctx.token, { template: true });
});

cmd("add-custom-field-to-project", "Add custom field to project", ["projectId", "customFieldId"], async (ctx, args) => {
  const cf = await doGet(ctx.base, `${API.CUSTOM_FIELDS}/customFields/${enc(args.customFieldId)}?fields=id,fieldType(valueType)`, ctx.token);
  const vt = (cf.fieldType?.valueType || "").toLowerCase();
  const type =
    vt.startsWith("enum")    ? "EnumProjectCustomField" :
    vt.startsWith("state")   ? "StateProjectCustomField" :
    vt.startsWith("version") ? "VersionProjectCustomField" :
    vt.startsWith("build")   ? "BuildProjectCustomField" :
    vt.startsWith("owned")   ? "OwnedProjectCustomField" :
    vt.startsWith("user")    ? "UserProjectCustomField" :
    vt.startsWith("group")   ? "GroupProjectCustomField" :
    "SimpleProjectCustomField";
  return doPost(ctx.base, `${API.PROJECTS}/${enc(args.projectId)}/fields?fields=id,field(id,name)`, ctx.token, {
    $type: type,
    field: { id: args.customFieldId },
  });
});

cmd("get-project-custom-fields", "List custom fields attached to a project", ["projectId"], async (ctx, args) => {
  return doGet(ctx.base, `${API.PROJECTS}/${enc(args.projectId)}/fields?fields=id,field(id,name),bundle(id,name)&$top=-1`, ctx.token);
});

cmd("set-time-tracking-settings", "Enable/configure time tracking for project", ["projectId", "?enabled", "?spentTimeField", "?estimationField"], async (ctx, args) => {
  const body = { enabled: parseBool(args.enabled) };
  if (args.spentTimeField) body.spentTimeField = { field: { name: args.spentTimeField } };
  if (args.estimationField) body.estimationField = { field: { name: args.estimationField } };
  return doPost(ctx.base, `${API.PROJECTS}/${enc(args.projectId)}/timeTrackingSettings?fields=enabled`, ctx.token, body);
});

cmd("add-user-to-project-team", "Add user to project team", ["projectShortName", "userId"], async (ctx, args) => {
  const projects = await doGet(ctx.base, `${API.PROJECTS}?query=${enc(args.projectShortName)}&fields=id,shortName&$top=10`, ctx.token);
  const proj = findByShortName(projects, args.projectShortName, "Project");
  return doPost(ctx.base, `${API.PROJECTS}/${proj.id}/team?fields=id`, ctx.token, { users: [{ id: args.userId }] });
});

cmd("add-group-to-project-team", "Add group to project team", ["projectShortName", "groupName"], async (ctx, args) => {
  const projects = await doGet(ctx.base, `${API.PROJECTS}?query=${enc(args.projectShortName)}&fields=id,shortName&$top=10`, ctx.token);
  const proj = findByShortName(projects, args.projectShortName, "Project");
  const groups = await doGet(ctx.base, `${API.GROUPS}?fields=id,name,$type&$top=-1&query=${enc(args.groupName)}`, ctx.token);
  const group = findByName(groups.filter(g => g.$type !== "ProjectTeam"), args.groupName, "Group");
  return doPost(ctx.base, `${API.PROJECTS}/${proj.id}/team?fields=id`, ctx.token, { groups: [{ id: group.id }] });
});

cmd("attach-workflow-to-project", "Attach workflow to project", ["workflowId", "projectId"], async (ctx, args) => {
  return doPost(ctx.base, `${API.PROJECTS}/${enc(args.projectId)}/workflows?fields=id`, ctx.token, { workflow: { id: args.workflowId } });
});

// ── Users ─────────────────────────────────────────────────────────────────────

cmd("add-user", "Create user", ["login", "fullName", "email", "?password", "?userType"], async (ctx, args) => {
  const userType = args.userType || "STANDARD_USER";
  if (!USER_TYPES.includes(userType)) throw new Error(`Invalid userType '${userType}'. Must be one of: ${USER_TYPES.join(", ")}`);
  const user = await doPost(ctx.base, `${API.USERS}?fields=id,ringId,login,fullName,email,isEmailVerified,guest,banned,userType(id,name)`, ctx.token, {
    login: args.login,
    fullName: args.fullName,
    email: args.email,
    password: args.password || undefined,
    isEmailVerified: true,
    userType: { id: userType },
  });
  // Wait until the user is searchable (Hub sync complete), mirroring Kotlin's waitForUserToBeFound.
  const deadline = Date.now() + 10000;
  while (Date.now() < deadline) {
    const users = await doGet(ctx.base, `${API.USERS}?query=${enc(args.login)}&fields=id,login&$top=10`, ctx.token);
    if (Array.isArray(users) && users.some(u => u.login === args.login)) break;
    await new Promise(r => setTimeout(r, 300));
  }
  return user;
});

cmd("get-users", "List/search users", ["?query"], async (ctx, args) => {
  const q = args.query ? `&query=${enc(args.query)}` : "";
  return doGet(ctx.base, `${API.USERS}?fields=id,login,fullName,email&$top=100${q}`, ctx.token);
});

cmd("get-user", "Get user by login", ["login"], async (ctx, args) => {
  const users = await doGet(ctx.base, `${API.USERS}?query=${enc(args.login)}&fields=id,login,fullName,email&$top=10`, ctx.token);
  return users.find(u => u.login === args.login) || null;
});

cmd("delete-user", "Delete user", ["userId", "?successor"], async (ctx, args) => {
  let successorId = args.successor;
  if (!successorId) {
    const users = await doGet(ctx.base, `${API.USERS}?query=guest&fields=id,login&$top=10`, ctx.token);
    const guest = users.find(u => u.login === "guest");
    successorId = guest?.id;
  }
  const qs = successorId ? `?successor=${enc(successorId)}` : "";
  const deadline = Date.now() + 10000;
  let lastErr;
  while (Date.now() < deadline) {
    try {
      return await doDelete(ctx.base, `${API.USERS}/${enc(args.userId)}${qs}`, ctx.token);
    } catch (e) {
      if (!e.message.includes("not mapped to hub")) throw e;
      lastErr = e;
      await new Promise(r => setTimeout(r, 500));
    }
  }
  throw lastErr;
});

cmd("ban-user", "Ban user", ["userId"], async (ctx, args) => {
  return doPost(ctx.base, `${API.USERS}/${enc(args.userId)}`, ctx.token, { banned: true });
});

cmd("unban-user", "Unban user", ["userId"], async (ctx, args) => {
  return doPost(ctx.base, `${API.USERS}/${enc(args.userId)}`, ctx.token, { banned: false });
});

cmd("get-current-user", "Get current authenticated user", [], async (ctx) => {
  return doGet(ctx.base, `${API.USERS_ME}?fields=id,login,fullName,email`, ctx.token);
});

// ── User Profile Settings ─────────────────────────────────────────────────────

cmd("set-appearance-setting", "Set user appearance setting (key-value)", ["?userId", "setting", "value"], async (ctx, args) => {
  const userId = parseUserIdOrMe(args.userId);
  const val = args.value === "true" ? true : args.value === "false" ? false : args.value;
  return doPost(ctx.base, `${API.USERS}/${enc(userId)}/profiles/appearance`, ctx.token,
    { [args.setting]: val });
});

cmd("set-appearance-json", "Set user appearance setting (raw JSON body)", ["?userId", "body"], async (ctx, args) => {
  const userId = parseUserIdOrMe(args.userId);
  return doPost(ctx.base, `${API.USERS}/${enc(userId)}/profiles/appearance`, ctx.token, JSON.parse(args.body));
});

cmd("set-profile-tips", "Set user profile tips (comma-separated key=value pairs)", ["?userId", "tips"], async (ctx, args) => {
  const userId = parseUserIdOrMe(args.userId);
  const body = {};
  for (const pair of args.tips.split(",")) {
    const [k, v] = pair.split("=").map(s => s.trim());
    body[k] = v === "true";
  }
  return doPost(ctx.base, `${API.USERS}/${enc(userId)}/profiles/tips`, ctx.token, body);
});

cmd("set-profile-property", "Set user profile property (key-value store)", ["?userId", "key", "value"], async (ctx, args) => {
  const userId = parseUserIdOrMe(args.userId);
  const propsUrl = `${API.USERS}/${enc(userId)}/profiles/propertiesProfile/properties?$top=-1&fields=key,value`;
  const list = await doGet(ctx.base, `${propsUrl}&query=${enc(args.key)}`, ctx.token);
  const existing = Array.isArray(list) ? list.find(p => p.key === args.key) : null;
  if (!existing || existing.value !== args.value) {
    return doPost(ctx.base, propsUrl, ctx.token, { key: args.key, value: args.value });
  }
  return existing;
});

cmd("set-issues-list-setting", "Set user issues list setting (key-value)", ["?userId", "setting", "value"], async (ctx, args) => {
  const userId = parseUserIdOrMe(args.userId);
  const val = args.value === "true" ? true : args.value === "false" ? false : args.value;
  return doPost(ctx.base, `${API.USERS}/${enc(userId)}/profiles/issuesList`, ctx.token,
    { [args.setting]: val });
});

cmd("set-issues-list-json", "Set user issues list setting (raw JSON body)", ["?userId", "body"], async (ctx, args) => {
  const userId = parseUserIdOrMe(args.userId);
  return doPost(ctx.base, `${API.USERS}/${enc(userId)}/profiles/issuesList`, ctx.token, JSON.parse(args.body));
});

cmd("set-articles-profile", "Set user articles profile setting", ["?userId", "body"], async (ctx, args) => {
  const userId = parseUserIdOrMe(args.userId);
  return doPost(ctx.base, `${API.USERS}/${enc(userId)}/profiles/articles`, ctx.token, JSON.parse(args.body));
});

cmd("set-general-profile", "Set user general profile setting (raw JSON body)", ["?userId", "body"], async (ctx, args) => {
  const userId = parseUserIdOrMe(args.userId);
  return doPost(ctx.base, `${API.USERS}/${enc(userId)}/profiles/general`, ctx.token, JSON.parse(args.body));
});

cmd("set-user-locale", "Set user locale", ["?userId", "locale"], async (ctx, args) => {
  const userId = parseUserIdOrMe(args.userId);
  return doPost(ctx.base, `${API.USERS}/${enc(userId)}/profiles/general`, ctx.token,
    { locale: { id: args.locale } });
});

cmd("get-user-timezone", "Get user timezone", ["?userId"], async (ctx, args) => {
  const userId = parseUserIdOrMe(args.userId);
  return doGet(ctx.base, `${API.USERS}/${enc(userId)}/profiles/general?fields=timezone(id,presentation)`, ctx.token);
});

cmd("set-user-timezone", "Set user timezone", ["?userId", "timezoneId"], async (ctx, args) => {
  const userId = parseUserIdOrMe(args.userId);
  return doPost(ctx.base, `${API.USERS}/${enc(userId)}/profiles/general`, ctx.token,
    { timezone: { id: args.timezoneId } });
});

cmd("set-notifications-profile", "Set user notifications profile", ["?userId", "body"], async (ctx, args) => {
  const userId = parseUserIdOrMe(args.userId);
  return doPost(ctx.base, `${API.USERS}/${enc(userId)}/profiles/notifications`, ctx.token, JSON.parse(args.body));
});

cmd("set-user-grazie-profile", "Set user Grazie (AI) profile settings", ["?userId", "body"], async (ctx, args) => {
  const userId = parseUserIdOrMe(args.userId);
  return doPost(ctx.base, `${API.USERS}/${enc(userId)}/profiles/grazie`, ctx.token, JSON.parse(args.body));
});

cmd("get-user-issue-drafts", "Get user issue drafts", ["?userId"], async (ctx, args) => {
  const userId = parseUserIdOrMe(args.userId);
  return doGet(ctx.base, `${API.USERS}/${enc(userId)}/drafts?fields=id,summary,created`, ctx.token);
});

cmd("delete-all-user-drafts", "Delete all user issue drafts", ["?userId"], async (ctx, args) => {
  const userId = parseUserIdOrMe(args.userId);
  return doPut(ctx.base, `${API.USERS}/${enc(userId)}/drafts`, ctx.token, []);
});

cmd("get-user-article-drafts", "Get user article drafts", ["?userId"], async (ctx, args) => {
  const userId = parseUserIdOrMe(args.userId);
  return doGet(ctx.base, `${API.USERS}/${enc(userId)}/articleDrafts?$top=-1&fields=id,summary`, ctx.token);
});

cmd("pin-project-for-user", "Pin project for user", ["?userId", "projectId"], async (ctx, args) => {
  const userId = parseUserIdOrMe(args.userId);
  return doPost(ctx.base, `${API.USERS}/${enc(userId)}/pinnedProjects`, ctx.token, { id: args.projectId });
});

// ── Groups ────────────────────────────────────────────────────────────────────

cmd("get-groups", "List/search groups", ["?query"], async (ctx, args) => {
  const q = args.query ? `&query=${enc(args.query)}` : "";
  return doGet(ctx.base, `${API.GROUPS}?fields=id,name&$top=-1${q}`, ctx.token);
});

cmd("get-group", "Get group by name", ["groupName"], async (ctx, args) => {
  const groups = await doGet(ctx.base, `${API.GROUPS}?fields=id,name&$top=-1&query=${enc(args.groupName)}`, ctx.token);
  return groups.find(g => g.name === args.groupName) || null;
});

cmd("add-group", "Create a group", ["name"], async (ctx, args) => {
  return doPost(ctx.base, `${API.GROUPS}?fields=id,name`, ctx.token, { name: args.name });
});

cmd("delete-group", "Delete group", ["groupName", "?successor"], async (ctx, args) => {
  const groups = await doGet(ctx.base, `${API.GROUPS}?fields=id,name&$top=-1&query=${enc(args.groupName)}`, ctx.token);
  const group = findByName(groups, args.groupName, "Group");
  // YouTrack always requires a successor group; default to "Registered Users" (6-0)
  const successor = args.successor || "6-0";
  return doDelete(ctx.base, `${API.GROUPS}/${group.id}?successor=${enc(successor)}`, ctx.token);
});

// ── Roles ─────────────────────────────────────────────────────────────────────

cmd("get-roles", "List all roles", [], async (ctx) => {
  return doGet(ctx.base, `${API.ROLES}?fields=id,name,description&$top=-1`, ctx.token);
});

cmd("add-role", "Create role", ["name", "?description", "?permissions"], async (ctx, args) => {
  const body = {
    name: args.name,
    description: args.description || "",
  };
  if (args.permissions) {
    body.permissions = parseIds(args.permissions);
  }
  return doPost(ctx.base, `${API.ROLES}?fields=id,name`, ctx.token, body);
});

cmd("update-role", "Update role permissions", ["roleId", "permissions"], async (ctx, args) => {
  const body = {
    permissions: parseIds(args.permissions)
  };
  return doPost(ctx.base, `${API.ROLES}/${enc(args.roleId)}?fields=id,name`, ctx.token, body);
});

cmd("grant-role-to-user", "Grant role to user (project or global scope)", ["userId", "roleName", "?projectName"], async (ctx, args) => {
  const roles = await doGet(ctx.base, `${API.ROLES}?fields=id,name&$top=-1`, ctx.token);
  const role = findByName(roles, args.roleName, "Role");
  let scope;
  if (args.projectName) {
    const projects = await doGet(ctx.base, `${API.PROJECTS}?query=${enc(args.projectName)}&fields=id,name&$top=10`, ctx.token);
    const project = findByName(projects, args.projectName, "Project");
    scope = { $type: "ProjectScope", project: { id: project.id } };
  } else {
    scope = { $type: "GlobalScope" };
  }
  return doPost(ctx.base, API.ASSIGNED_ROLES, ctx.token, {
    holder: { id: args.userId, $type: "User" },
    role: { id: role.id },
    scope,
  });
});

cmd("delete-role", "Delete a role", ["roleId"], async (ctx, args) => {
  return doDelete(ctx.base, `${API.ROLES}/${enc(args.roleId)}`, ctx.token);
});

cmd("grant-role-to-group", "Grant role to group (project or global scope)", ["groupName", "roleName", "?projectName"], async (ctx, args) => {
  const roles = await doGet(ctx.base, `${API.ROLES}?fields=id,name&$top=-1`, ctx.token);
  const role = findByName(roles, args.roleName, "Role");
  const groups = await doGet(ctx.base, `${API.GROUPS}?fields=id,name,$type&$top=-1&query=${enc(args.groupName)}`, ctx.token);
  const group = findByName(groups.filter(g => g.$type !== "ProjectTeam"), args.groupName, "Group");
  let scope;
  if (args.projectName) {
    const projects = await doGet(ctx.base, `${API.PROJECTS}?query=${enc(args.projectName)}&fields=id,name&$top=10`, ctx.token);
    const project = findByName(projects, args.projectName, "Project");
    scope = { $type: "ProjectScope", project: { id: project.id } };
  } else {
    scope = { $type: "GlobalScope" };
  }
  return doPost(ctx.base, API.ASSIGNED_ROLES, ctx.token, {
    holder: { id: group.id, $type: group.$type },
    role: { id: role.id },
    scope,
  });
});

cmd("get-permissions", "List all permissions", [], async (ctx) => {
  const fields = "id,name,description,permissionEntityType,operation,isGlobal,dependentPermissions(id,name),impliedPermissions(id,name)";
  return doGet(ctx.base, `${API.PERMISSIONS}?fields=${fields}&$top=-1`, ctx.token);
});

// ── Dashboards ────────────────────────────────────────────────────────────────

cmd("add-dashboard", "Create dashboard", ["name"], async (ctx, args) => {
  return doPost(ctx.base, `${API.DASHBOARDS}?fields=id,name`, ctx.token, { name: args.name });
});

cmd("get-dashboards", "List dashboards", [], async (ctx) => {
  return doGet(ctx.base, `${API.DASHBOARDS}?fields=id,name&$top=-1`, ctx.token);
});

cmd("delete-dashboard", "Delete dashboard", ["dashboardId"], async (ctx, args) => {
  return doDelete(ctx.base, `${API.DASHBOARDS}/${enc(args.dashboardId)}`, ctx.token);
});

cmd("update-dashboard-name", "Rename a dashboard", ["dashboardId", "newName"], async (ctx, args) => {
  return doPost(ctx.base, `${API.DASHBOARDS}/${enc(args.dashboardId)}?fields=id,name`, ctx.token, { name: args.newName });
});

// ── Agile Boards ──────────────────────────────────────────────────────────────

cmd("add-agile-board", "Create agile board", ["name", "?template", "?projectIds"], async (ctx, args) => {
  const template = args.template || "scrum";
  const body = { name: args.name };
  if (args.projectIds) body.projects = parseIds(args.projectIds);
  return doPost(ctx.base, `${API.AGILES}?template=${enc(template)}&fields=id,name`, ctx.token, body);
});

cmd("get-agile-boards", "List agile boards", [], async (ctx) => {
  return doGet(ctx.base, `${API.AGILES}?fields=id,name&$top=-1`, ctx.token);
});

cmd("get-agile-board", "Get agile board by ID", ["boardId"], async (ctx, args) => {
  return doGet(ctx.base, `${API.AGILES}/${enc(args.boardId)}?fields=id,name,projects(id,shortName),sprints(id,name)`, ctx.token);
});

cmd("delete-agile-board", "Delete agile board", ["boardId"], async (ctx, args) => {
  return doDelete(ctx.base, `${API.AGILES}/${enc(args.boardId)}`, ctx.token);
});

cmd("add-sprint", "Create sprint", ["boardId", "name", "startDate", "endDate", "?goal"], async (ctx, args) => {
  return doPost(ctx.base, `${API.AGILES}/${enc(args.boardId)}/sprints?fields=id,name,start,finish,goal`, ctx.token, {
    name: args.name,
    start: parseDate(args.startDate),
    finish: parseDate(args.endDate),
    goal: args.goal || "",
  });
});

// ── Tags ──────────────────────────────────────────────────────────────────────

cmd("tag-issue", "Tag an issue", ["issueId", "tagId"], async (ctx, args) => {
  return doPost(ctx.base, `${API.ISSUES}/${enc(args.issueId)}/tags?fields=id`, ctx.token, { $type: "Tag", id: args.tagId });
});

cmd("delete-tag", "Delete tag", ["tagId"], async (ctx, args) => {
  return doDelete(ctx.base, `${API.TAGS}/${enc(args.tagId)}`, ctx.token);
});

cmd("add-tag", "Create a tag with sharing settings", ["name", "ownerId", "?colorId", "?isShareable", "?isUpdatable", "?untagOnResolve", "?pinned", "?pinnedByDefault"], async (ctx, args) => {
  const sharingSettings = { permittedGroups: [], permittedUsers: [] };
  const body = {
    name: args.name,
    owner: { id: args.ownerId },
    isShareable: args.isShareable !== undefined ? parseBool(args.isShareable) : true,
    isUpdatable: args.isUpdatable !== undefined ? parseBool(args.isUpdatable) : true,
    untagOnResolve: args.untagOnResolve !== undefined ? parseBool(args.untagOnResolve) : false,
    pinned: args.pinned !== undefined ? parseBool(args.pinned) : false,
    pinnedByDefault: args.pinnedByDefault !== undefined ? parseBool(args.pinnedByDefault) : false,
    readSharingSettings: sharingSettings,
    updateSharingSettings: sharingSettings,
    tagSharingSettings: sharingSettings,
  };
  if (args.colorId) body.color = { id: args.colorId };
  return doPost(ctx.base, `${API.TAGS}?fields=id,name,owner(id,login),color(id),isShareable,isUpdatable,pinned`, ctx.token, body);
});

// ── Work Items ────────────────────────────────────────────────────────────────

cmd("add-work-item", "Add work item to issue", ["issueId", "minutes", "?workType", "?text", "?authorLogin", "?date"], async (ctx, args) => {
  const body = { duration: { minutes: parseInt(args.minutes) } };
  if (args.text) body.text = args.text;
  if (args.date) body.date = parseInt(args.date);
  if (args.workType) {
    const types = await doGet(ctx.base, `${API.WORK_ITEM_TYPES}?fields=id,name`, ctx.token);
    const t = types.find(t => t.name === args.workType);
    if (t) body.type = { id: t.id };
  }
  if (args.authorLogin) {
    const u = await resolveUserRef(ctx.base, ctx.token, args.authorLogin);
    body.author = { id: u.id };
  }
  return doPost(ctx.base, `${API.ISSUES}/${enc(args.issueId)}/timeTracking/workItems`, ctx.token, body);
});

// ── Workflows ─────────────────────────────────────────────────────────────────

cmd("get-workflows", "List all workflows", [], async (ctx) => {
  return doGet(ctx.base, `${API.WORKFLOWS}?$top=-1&fields=id,name,title,rules(id,title,name)`, ctx.token);
});

cmd("delete-workflow", "Delete workflow", ["workflowId"], async (ctx, args) => {
  return doDelete(ctx.base, `${API.WORKFLOWS}/${enc(args.workflowId)}`, ctx.token);
});

cmd("add-workflow", "Create workflow", ["name", "title"], async (ctx, args) => {
  return doPost(ctx.base, `${API.WORKFLOWS}?fields=id,name,title`, ctx.token, { name: args.name, title: args.title });
});

cmd("add-workflow-rule", "Create rule in workflow", ["workflowId", "name", "script"], async (ctx, args) => {
  return doPost(ctx.base, `${API.WORKFLOWS}/${enc(args.workflowId)}/rules?fields=id,name,title`, ctx.token, { name: args.name, script: args.script });
});

cmd("restore-default-workflows", "Restore default workflows", [], async (ctx) => {
  return doPost(ctx.base, `${API.WORKFLOWS}/restoreStatus?fields=isDefault`, ctx.token, { isDefault: true });
});

// ── Global Settings ───────────────────────────────────────────────────────────

cmd("get-global-settings", "Get global settings", [], async (ctx) => {
  return doGet(ctx.base, `${API.GLOBAL_SETTINGS}?fields=id,baseUrl,maxUploadFileSize,maxExportItems,ringUrl`, ctx.token);
});

cmd("get-frontend-config", "Get YouTrack frontend configuration", [], async (ctx) => {
  return doGet(ctx.base, `/api/config`, ctx.token);
});

cmd("enable-feature-flag", "Toggle system-wide feature flag", ["feature", "?enabled"], async (ctx, args) => {
  return doPost(ctx.base, `${API.FEATURE_FLAGS}/${enc(args.feature)}`, ctx.token, { id: args.feature, enabled: parseBool(args.enabled) });
});

cmd("set-license", "Set YouTrack license", ["username", "license"], async (ctx, args) => {
  return doPost(ctx.base, API.GLOBAL_SETTINGS, ctx.token, { license: { username: args.username, license: args.license } });
});

cmd("set-email-settings", "Configure email server settings", ["serverAddress", "port", "fromAddress", "?replyToAddress"], async (ctx, args) => {
  return doPost(ctx.base, `${API.GLOBAL_SETTINGS}/notificationSettings?fields=emailSettings(isEnabled,from,replyTo)`, ctx.token, {
    emailSettings: { isEnabled: true, anonymous: true, mailProtocol: { id: "SMTP" }, host: args.serverAddress, port: parseInt(args.port), from: args.fromAddress, replyTo: args.replyToAddress || undefined },
  });
});

cmd("disable-email-settings", "Disable email server settings", [], async (ctx) => {
  return doPost(ctx.base, `${API.GLOBAL_SETTINGS}/notificationSettings?fields=emailSettings(isEnabled,from,replyTo)`, ctx.token, {
    emailSettings: { isEnabled: false },
  });
});

cmd("enable-global-banner", "Enable global banner message", ["message"], async (ctx, args) => {
  return doPost(ctx.base, API.GLOBAL_SETTINGS, ctx.token, { appearanceSettings: { globalBannerEnabled: true, globalBanner: args.message } });
});

cmd("disable-global-banner", "Disable global banner", [], async (ctx) => {
  return doPost(ctx.base, API.GLOBAL_SETTINGS, ctx.token, { appearanceSettings: { globalBannerEnabled: false } });
});

cmd("set-reactions-enabled", "Enable/disable reactions", ["enabled"], async (ctx, args) => {
  return doPost(ctx.base, API.GLOBAL_SETTINGS, ctx.token, { appearanceSettings: { reactionsEnabled: parseBool(args.enabled) } });
});

cmd("switch-jetbrains-ai", "Enable/disable JetBrains AI", ["enabled"], async (ctx, args) => {
  return doPost(ctx.base, API.GRAZIE, ctx.token, { enabled: parseBool(args.enabled) });
});

cmd("switch-text-completion", "Enable/disable AI text completion", ["enabled"], async (ctx, args) => {
  return doPost(ctx.base, API.GRAZIE, ctx.token, { enableTextCompletion: parseBool(args.enabled) });
});

cmd("switch-grammar-correction", "Enable/disable AI grammar correction", ["enabled"], async (ctx, args) => {
  return doPost(ctx.base, API.GRAZIE, ctx.token, { enableSpellChecker: parseBool(args.enabled) });
});

// ── Saved Searches ────────────────────────────────────────────────────────────

cmd("add-saved-search", "Create saved search", ["query", "name"], async (ctx, args) => {
  return doPost(ctx.base, `${API.SAVED_QUERIES}?fields=id,query,name`, ctx.token, { query: args.query, name: args.name });
});

cmd("share-saved-search", "Share saved search with users/groups", ["searchId", "?users", "?groups"], async (ctx, args) => {
  const permittedUsers = args.users ? await Promise.all(args.users.split(",").map(async l => {
    const users = await doGet(ctx.base, `${API.USERS}?query=${enc(l.trim())}&fields=id,login&$top=10`, ctx.token);
    return findByLogin(users, l.trim(), "User");
  })) : [];
  const permittedGroups = args.groups ? await Promise.all(args.groups.split(",").map(async n => {
    const groups = await doGet(ctx.base, `${API.GROUPS}?fields=id,name&$top=-1&query=${enc(n.trim())}`, ctx.token);
    return findByName(groups, n.trim(), "Group");
  })) : [];
  return doPost(ctx.base, `${API.SAVED_QUERIES}/${enc(args.searchId)}?fields=id,query,name,owner`, ctx.token, {
    id: args.searchId, isShareable: true,
    readSharingSettings: { permittedGroups, permittedUsers },
  });
});

// ── Reports ───────────────────────────────────────────────────────────────────

cmd("delete-report", "Delete report", ["reportId"], async (ctx, args) => {
  return doDelete(ctx.base, `${API.REPORTS}/${enc(args.reportId)}`, ctx.token);
});

// ── Apps ──────────────────────────────────────────────────────────────────────

cmd("get-apps", "List all apps", [], async (ctx) => {
  return doGet(ctx.base, `${API.APPS}?$top=-1&fields=id,name,title,version`, ctx.token);
});

cmd("delete-app", "Delete app", ["appId"], async (ctx, args) => {
  return doDelete(ctx.base, `${API.APPS}/${enc(args.appId)}`, ctx.token);
});

cmd("deactivate-app", "Deactivate app globally", ["appId"], async (ctx, args) => {
  return doPost(ctx.base, `${API.APPS}/${enc(args.appId)}?fields=id,name,globalConfig(enabled)`, ctx.token, { globalConfig: { enabled: false } });
});

cmd("get-plugins", "List plugin usages", [], async (ctx) => {
  return doGet(ctx.base, `${API.PLUGIN_USAGES}?fields=id,plugin(name),state,continuous,enabled`, ctx.token);
});

cmd("delete-plugin", "Remove plugin usage", ["pluginId"], async (ctx, args) => {
  return doDelete(ctx.base, `${API.PLUGIN_USAGES}/${enc(args.pluginId)}`, ctx.token);
});

// ── VCS ───────────────────────────────────────────────────────────────────────

cmd("get-vcs-servers", "List VCS servers", [], async (ctx) => {
  return doGet(ctx.base, `${API.VCS_SERVERS}?fields=id,type,url`, ctx.token);
});

cmd("delete-vcs-server", "Delete VCS server", ["serverId"], async (ctx, args) => {
  return doDelete(ctx.base, `${API.VCS_SERVERS}/${enc(args.serverId)}`, ctx.token);
});

cmd("get-vcs-configurations", "List VCS configurations for a server", ["serverId"], async (ctx, args) => {
  return doGet(ctx.base, `${API.VCS_SERVERS}/${enc(args.serverId)}/changesProcessors?fields=id,path,executeImport`, ctx.token);
});

cmd("get-vcs-hosting-processors", "List VCS hosting processors", [], async (ctx) => {
  return doGet(ctx.base, `${API.VCS_PROCESSORS}?$top=-1&fields=id,path,server($type,id,url),projectRepo,enabled,project(id,name)`, ctx.token);
});

cmd("delete-vcs-hosting-processor", "Remove VCS hosting processor", ["processorId"], async (ctx, args) => {
  return doDelete(ctx.base, `${API.VCS_PROCESSORS}/${enc(args.processorId)}`, ctx.token);
});

// ── Mailbox ───────────────────────────────────────────────────────────────────

cmd("get-mailbox-servers", "List mailbox servers", [], async (ctx) => {
  return doGet(ctx.base, `${API.MAILBOXES}?$top=-1&fields=id,login,host,port`, ctx.token);
});

cmd("delete-mailbox-server", "Delete mailbox server", ["mailboxId"], async (ctx, args) => {
  return doDelete(ctx.base, `${API.MAILBOXES}/${enc(args.mailboxId)}`, ctx.token);
});

cmd("add-mailbox-server", "Create mailbox server", ["protocol", "host", "login", "password", "?port"], async (ctx, args) => {
  return doPost(ctx.base, `${API.MAILBOXES}?$top=-1&fields=id`, ctx.token, {
    exchangeCompatibilityMode: false,
    host: args.host,
    port: parseInt(args.port || "143"),
    login: args.login,
    password: args.password,
    protocol: { id: args.protocol },
  });
});

cmd("add-mailbox-rule", "Create mailbox rule", ["mailboxId", "projectId", "?name", "?disabled"], async (ctx, args) => {
  return doPost(ctx.base, `${API.MAILBOX_RULES}?$top=-1&fields=id,disabled,mailbox`, ctx.token, {
    mailbox: { id: args.mailboxId },
    project: { id: args.projectId },
    disabled: args.disabled === "true",
    name: args.name || undefined,
  });
});

// ── Organizations ─────────────────────────────────────────────────────────────

cmd("get-organizations", "List organizations", [], async (ctx) => {
  return doGet(ctx.base, `${API.ORGANIZATIONS}?fields=id,name&$top=-1`, ctx.token);
});

cmd("delete-organization", "Delete organization", ["organizationId"], async (ctx, args) => {
  return doDelete(ctx.base, `${API.ORGANIZATIONS}/${enc(args.organizationId)}`, ctx.token);
});

cmd("add-organization", "Create organization", ["name"], async (ctx, args) => {
  return doPost(ctx.base, `${API.ORGANIZATIONS}?fields=id,name`, ctx.token, { name: args.name });
});

// ── Attachments ───────────────────────────────────────────────────────────────

cmd("get-attachments", "List issue attachments", ["issueId"], async (ctx, args) => {
  return doGet(ctx.base, `${API.ISSUES}/${enc(args.issueId)}/attachments?fields=id,name,url,thumbnailURL,mimeType,extension,created`, ctx.token);
});

cmd("add-attachment", "Upload file attachment to an issue", ["issueId", "file", "?name"], async (ctx, args) => {
  const fileName = args.name || basename(args.file);
  const mimeType = MIME_TYPES[extname(fileName).toLowerCase()] || "application/octet-stream";
  return doMultipart(ctx.base, `${API.ISSUES}/${enc(args.issueId)}/attachments?fields=id,name,url,thumbnailURL,mimeType`, ctx.token, args.file, fileName, mimeType);
});

cmd("delete-attachment", "Delete attachment", ["issueId", "attachmentId"], async (ctx, args) => {
  return doDelete(ctx.base, `${API.ISSUES}/${enc(args.issueId)}/attachments/${enc(args.attachmentId)}`, ctx.token);
});

cmd("add-article-attachment", "Upload file attachment to an article", ["articleId", "file", "?name"], async (ctx, args) => {
  const fileName = args.name || basename(args.file);
  const mimeType = MIME_TYPES[extname(fileName).toLowerCase()] || "application/octet-stream";
  return doMultipart(ctx.base, `${API.ARTICLES}/${enc(args.articleId)}/attachments?fields=id,name,url,thumbnailURL,mimeType`, ctx.token, args.file, fileName, mimeType);
});

cmd("get-article-attachments", "List article attachments", ["articleId"], async (ctx, args) => {
  return doGet(ctx.base, `${API.ARTICLES}/${enc(args.articleId)}/attachments?fields=id,name,url,thumbnailURL,mimeType,extension,created`, ctx.token);
});

// ── Reactions ─────────────────────────────────────────────────────────────────

cmd("add-reaction", "Add reaction to comment", ["issueId", "commentId", "reaction"], async (ctx, args) => {
  return doPost(ctx.base, `${API.ISSUES}/${enc(args.issueId)}/comments/${enc(args.commentId)}/reactions`, ctx.token, { reaction: args.reaction });
});

cmd("delete-reaction", "Remove reaction from comment", ["issueId", "commentId", "reactionId"], async (ctx, args) => {
  return doDelete(ctx.base, `${API.ISSUES}/${enc(args.issueId)}/comments/${enc(args.commentId)}/reactions/${enc(args.reactionId)}`, ctx.token);
});

// ── Gantt ─────────────────────────────────────────────────────────────────────

cmd("get-gantt-charts", "List Gantt charts", [], async (ctx) => {
  return doGet(ctx.base, `${API.GANTT}?fields=id,name&$top=-1`, ctx.token);
});

cmd("delete-gantt-chart", "Delete Gantt chart", ["ganttId"], async (ctx, args) => {
  return doDelete(ctx.base, `${API.GANTT}/${enc(args.ganttId)}`, ctx.token);
});

// ── Canvas ────────────────────────────────────────────────────────────────────

cmd("get-canvas", "Get canvas by ID", ["canvasId"], async (ctx, args) => {
  return doGet(ctx.base, `${API.CANVAS}/${enc(args.canvasId)}?fields=id,name,owner(id,login)`, ctx.token);
});

cmd("delete-canvas", "Delete canvas", ["canvasId"], async (ctx, args) => {
  return doDelete(ctx.base, `${API.CANVAS}/${enc(args.canvasId)}`, ctx.token);
});

cmd("reload-canvas", "Reload canvas to sync frontend with latest data", ["canvasId"], async (ctx, args) => {
  return doPost(ctx.base, `${API.CANVAS}/${enc(args.canvasId)}?fields=id,name`, ctx.token, { reload: true });
});

cmd("add-canvas", "Create planning canvas", ["name", "ownerLogin"], async (ctx, args) => {
  const users = await doGet(ctx.base, `${API.USERS}?query=${enc(args.ownerLogin)}&fields=id,login&$top=10`, ctx.token);
  const user = findByLogin(users, args.ownerLogin, "User");
  return doPost(ctx.base, `${API.CANVAS}?fields=id,name`, ctx.token, { name: args.name, owner: { id: user.id } });
});

cmd("add-canvas-node", "Add card node to canvas", ["canvasId", "x", "y", "?label", "?issueId", "?articleId", "?nodeType", "?description"], async (ctx, args) => {
  const isTicket = args.issueId || (args.nodeType && args.nodeType.toLowerCase() === "ticketnode");
  const isArticle = args.articleId || (args.nodeType && args.nodeType.toLowerCase() === "articlenode");
  const resolvedType = isTicket ? "TicketNode" : isArticle ? "ArticleNode" : (args.nodeType || "CardNode");
  const body = { x: parseInt(args.x), y: parseInt(args.y), nodeType: { id: resolvedType } };
  if (isTicket && args.issueId) body.ticket = { id: args.issueId };
  if (isArticle && args.articleId) body.article = { id: args.articleId };
  if (args.label) body.label = args.label;
  if (args.description) body.description = args.description;
  return doPost(ctx.base, `${API.CANVAS}/${enc(args.canvasId)}/cards?fields=id,label,x,y,height,width,nodeType(id),ticket(id,idReadable,summary),article(id,idReadable,summary)`, ctx.token, body);
});

cmd("get-canvas-links", "List links in canvas", ["canvasId"], async (ctx, args) => {
  return doGet(ctx.base, `${API.CANVAS}/${enc(args.canvasId)}/links?fields=id,sourceCard(id,label),targetCard(id,label)`, ctx.token);
});

cmd("link-canvas-nodes", "Link two canvas nodes", ["canvasId", "sourceCardId", "targetCardId", "?linkType"], async (ctx, args) => {
  return doPost(ctx.base, `${API.CANVAS}/${enc(args.canvasId)}/links?fields=id,sourceCard(id),targetCard(id)`, ctx.token, {
    $type: "PlanningCanvasLink",
    type: { $type: "PlanningCanvasLinkType", id: args.linkType || "ONE_WAY" },
    sourceCard: { id: args.sourceCardId },
    targetCard: { id: args.targetCardId },
  });
});

// ── Custom Field Prototypes & Bundles ─────────────────────────────────────────

cmd("get-custom-field-prototypes", "List custom field prototypes", [], async (ctx) => {
  return doGet(ctx.base, `${API.CUSTOM_FIELDS}/customFields?fields=id,name,fieldType(id)`, ctx.token);
});

cmd("add-custom-field", "Create custom field prototype", ["name", "fieldTypeId"], async (ctx, args) => {
  return doPost(ctx.base, `${API.CUSTOM_FIELDS}/customFields?fields=id,name,fieldType(id)`, ctx.token, { name: args.name, fieldType: { id: args.fieldTypeId } });
});

cmd("delete-custom-field", "Delete custom field prototype", ["customFieldId"], async (ctx, args) => {
  return doDelete(ctx.base, `${API.CUSTOM_FIELDS}/customFields/${enc(args.customFieldId)}`, ctx.token);
});

const BUNDLE_TYPE_MAP = { enum: "EnumBundle", build: "BuildBundle", version: "VersionBundle" };
function bundleTypeName(t) {
  const n = BUNDLE_TYPE_MAP[t.replace(/Bundle$/i, "").toLowerCase()];
  if (!n) throw new Error(`Invalid bundleType '${t}'. Must be one of: enum, build, version`);
  return n;
}
function bundlePath(t) { return t.replace(/Bundle$/i, "").toLowerCase(); }

cmd("add-bundle-value", "Add value to enum bundle", ["bundleType", "bundleId", "value"], async (ctx, args) => {
  return doPost(ctx.base, `${API.CUSTOM_FIELDS}/bundles/${enc(bundlePath(args.bundleType))}/${enc(args.bundleId)}/values`, ctx.token, { name: args.value });
});

cmd("get-bundles", "List bundles of a given type", ["bundleType"], async (ctx, args) => {
  return doGet(ctx.base, `${API.CUSTOM_FIELDS}/bundles/${enc(bundlePath(args.bundleType))}?fields=id,name&$top=-1`, ctx.token);
});

cmd("create-bundle", "Create a custom field bundle", ["bundleType", "?name"], async (ctx, args) => {
  const body = { $type: bundleTypeName(args.bundleType) };
  if (args.name) body.name = args.name;
  return doPost(ctx.base, `${API.CUSTOM_FIELDS}/bundles/${enc(bundlePath(args.bundleType))}?fields=id,name`, ctx.token, body);
});

cmd("attach-bundle-to-field", "Attach a bundle to a custom field prototype", ["customFieldId", "bundleId", "bundleType"], async (ctx, args) => {
  return doPut(ctx.base, `${API.CUSTOM_FIELDS}/customFields/${enc(args.customFieldId)}/fieldDefaults/bundle?fields=id,name`, ctx.token, { id: args.bundleId, $type: bundleTypeName(args.bundleType) });
});

cmd("change-custom-field-privacy", "Set isPublic on a custom field prototype", ["customFieldId", "?isPublic"], async (ctx, args) => {
  const isPublic = args.isPublic !== undefined ? parseBool(args.isPublic) : false;
  return doPost(ctx.base, `${API.CUSTOM_FIELDS}/customFields/${enc(args.customFieldId)}?fields=id,fieldDefaults(isPublic)`, ctx.token, { fieldDefaults: { isPublic } });
});

// ── Work Time & Work Item Types ───────────────────────────────────────────────

cmd("get-work-time-settings", "Get work time settings", [], async (ctx) => {
  return doGet(ctx.base, `${API.WORK_TIME}?fields=id,daysAWeek,minutesADayPresentation,minutesADay,firstDayOfWeek,workDays`, ctx.token);
});

cmd("set-work-time-settings", "Set work time settings (hours in a day)", ["hoursInADay"], async (ctx, args) => {
  return doPost(ctx.base, `${API.WORK_TIME}?fields=id,daysAWeek,minutesADayPresentation,minutesADay`, ctx.token, { minutesADayPresentation: `${args.hoursInADay}h` });
});

cmd("get-work-item-types", "List work item types", [], async (ctx) => {
  return doGet(ctx.base, `${API.WORK_ITEM_TYPES}?fields=id,name`, ctx.token);
});

cmd("add-work-item-type", "Create work item type", ["name", "?autoAttach"], async (ctx, args) => {
  return doPost(ctx.base, `${API.WORK_ITEM_TYPES}?fields=id,name`, ctx.token, { name: args.name, autoAttach: args.autoAttach === "true" });
});

// ── Auth / Token ──────────────────────────────────────────────────────────────

cmd("get-token", "Get permanent token using login/password (basic auth)", ["login", "password", "?tokenName", "?scope", "?setEnv"], async (ctx, args) => {
  const base = ctx.base;
  const login = args.login;
  const password = args.password;
  const tokenName = args.tokenName || "cli-token";

  const me = await httpReqBasicAuth("GET", base, "/hub/api/rest/users/me?fields=id,login,name", login, password);
  if (!me || !me.id) throw new Error("Failed to authenticate — check login/password");

  const requestedScope = args.scope;
  const scopeIds = [];
  const services = await httpReqBasicAuth("GET", base, "/hub/api/rest/services?fields=id,name&$top=-1", login, password);
  const serviceList = Array.isArray(services) ? services : (services.services || []);

  if (requestedScope) {
    for (const name of requestedScope.split(",").map(s => s.trim())) {
      const svc = serviceList.find(s => s.name === name || s.id === name);
      if (svc) scopeIds.push({ id: svc.id });
      else throw new Error(`Service/scope '${name}' not found. Available: ${serviceList.map(s => s.name).join(", ")}`);
    }
  } else {
    const yt = serviceList.find(s => s.name === "YouTrack");
    if (yt) scopeIds.push({ id: yt.id });
    const hub = serviceList.find(s => s.name === "Hub");
    if (hub) scopeIds.push({ id: hub.id });
    if (scopeIds.length === 0 && serviceList.length > 0) scopeIds.push({ id: serviceList[0].id });
  }

  const tokenBody = { name: tokenName, scope: scopeIds };
  const result = await httpReqBasicAuth("POST", base,
    `/hub/api/rest/users/${enc(me.id)}/permanenttokens?fields=id,name,token,scope(id,name),creationTime`,
    login, password, tokenBody);

  if (args.setEnv === "true" || args.setEnv === "1" || args.setEnv === "yes") {
    const envFile = join(homedir(), ".youtrack-cli-env");
    const envContent = `YOUTRACK_BASE_URL=${base}\nYOUTRACK_TOKEN=${result.token}\n`;
    writeFileSync(envFile, envContent, { mode: 0o600 });
    result._envFile = envFile;
    result._hint = "Token saved. Source it: export $(cat ~/.youtrack-cli-env | xargs)";
  }

  return result;
});

// ── Raw API call ──────────────────────────────────────────────────────────────

cmd("raw-get", "Raw GET request", ["path"], async (ctx, args) => {
  return doGet(ctx.base, args.path, ctx.token);
});

cmd("raw-post", "Raw POST request with JSON body", ["path", "body"], async (ctx, args) => {
  return doPost(ctx.base, args.path, ctx.token, JSON.parse(args.body));
});

cmd("raw-delete", "Raw DELETE request", ["path"], async (ctx, args) => {
  return doDelete(ctx.base, args.path, ctx.token);
});

cmd("raw", "Authenticated API call with method, pagination and field selection",
  ["path", "?method", "?body", "?apiFields", "?top", "?paginate"],
  async (ctx, args) => {
    const method = (args.method || "GET").toUpperCase();
    if (process.env.YOUTRACK_RO === "1" && method !== "GET") {
      throw new Error("Read-only mode active (YOUTRACK_RO=1). Write operations are blocked.");
    }
    let path = args.path;
    if (args.apiFields) {
      path += (path.includes("?") ? "&" : "?") + `$fields=${enc(args.apiFields)}`;
    }
    const body = args.body ? JSON.parse(args.body) : null;
    if (!args.paginate || args.paginate === "false") {
      if (args.top) path += (path.includes("?") ? "&" : "?") + `$top=${args.top}`;
      return httpReq(method, ctx.base, path, ctx.token, body);
    }
    // Paginate: accumulate pages until a page is smaller than pageSize
    const pageSize = args.top ? parseInt(args.top) : 100;
    let skip = 0;
    const results = [];
    while (true) {
      const sep = path.includes("?") ? "&" : "?";
      const page = await httpReq("GET", ctx.base, `${path}${sep}$top=${pageSize}&$skip=${skip}`, ctx.token);
      if (!Array.isArray(page) || page.length === 0) break;
      results.push(...page);
      if (page.length < pageSize) break;
      skip += pageSize;
    }
    return results;
  }
);

// ── JSON mode (for programmatic use) ─────────────────────────────────────────

cmd("list-commands", "List all available commands", [], async () => {
  const entries = Object.entries(COMMANDS);
  if (!process.stdout.isTTY) {
    return entries.map(([name, c]) => ({ name, description: c.description, params: c.params }));
  }
  const maxName = Math.max(...entries.map(([n]) => n.length));
  const maxParams = Math.max(...entries.map(([, c]) => {
    return c.params.map(p => p.startsWith("?") ? `[--${p.slice(1)}]` : `--${p}`).join(" ").length;
  }));
  const lines = entries.map(([name, c]) => {
    const params = c.params.map(p => p.startsWith("?") ? `[--${p.slice(1)}]` : `--${p}`).join(" ");
    return `  ${name.padEnd(maxName + 2)}${params.padEnd(maxParams + 2)}${c.description}`;
  });
  return lines.join("\n");
});

// ── Main ──────────────────────────────────────────────────────────────────────

function printUsage() {
  console.log("YouTrack CLI — REST API tool for AI agents and developers\n");
  console.log("Usage: youtrack-cli <command> [--param value ...]\n");
  console.log("Environment:");
  console.log("  YOUTRACK_BASE_URL  YouTrack instance URL (required)");
  console.log("  YOUTRACK_TOKEN     Permanent token (required)");
  console.log("  YOUTRACK_RO=1      Read-only mode — blocks all write operations\n");
  console.log("Global options:");
  console.log("  --verbose, --debug  Log full HTTP requests/responses to stderr (auth redacted)");
  console.log("  --quiet, -q         Suppress output (useful for scripting side-effects)");
  console.log("  --fields <list>     Filter output to comma-separated fields (e.g. id,summary)");
  console.log("                      Use --fields ? to list available top-level fields");
  console.log("  --base-url URL      Override YOUTRACK_BASE_URL");
  console.log("  --token TOKEN       Override YOUTRACK_TOKEN\n");
  console.log("Run 'youtrack-cli list-commands' to see all available commands.");
}

async function main() {
  const argv = process.argv.slice(2);
  if (argv.length === 0 || argv[0] === "--help" || argv[0] === "-h") {
    printUsage();
    process.exit(0);
  }

  const commandName = argv[0];
  const command = COMMANDS[commandName];
  if (!command) {
    console.error(`Unknown command: ${commandName}\nRun with --help to see available commands.`);
    process.exit(1);
  }

  // Extract global boolean flags before --key value parsing
  let verbose = false, quiet = false;
  const filteredArgv = [];
  for (const arg of argv.slice(1)) {
    if (arg === "--verbose" || arg === "--debug") { verbose = true; continue; }
    if (arg === "--quiet" || arg === "-q") { quiet = true; continue; }
    filteredArgv.push(arg);
  }
  if (verbose) setVerbose(true);

  // Parse --key value pairs
  const parsed = {};
  for (let i = 0; i < filteredArgv.length; i++) {
    if (filteredArgv[i].startsWith("--") && i + 1 < filteredArgv.length) {
      const key = filteredArgv[i].slice(2).replace(/-([a-z])/g, (_, c) => c.toUpperCase());
      parsed[key] = filteredArgv[++i];
    }
  }

  // Load saved env from ~/.youtrack-cli-env if present
  const savedEnvFile = join(homedir(), ".youtrack-cli-env");
  if (existsSync(savedEnvFile)) {
    try {
      const lines = readFileSync(savedEnvFile, "utf-8").split("\n");
      for (const line of lines) {
        const eq = line.indexOf("=");
        if (eq > 0) {
          const k = line.slice(0, eq).trim();
          const v = line.slice(eq + 1).trim();
          if (!process.env[k]) process.env[k] = v;
        }
      }
    } catch { /* ignore */ }
  }

  const baseUrl = parsed.baseUrl || process.env.YOUTRACK_BASE_URL;
  const token = parsed.token || process.env.YOUTRACK_TOKEN;
  delete parsed.baseUrl;
  delete parsed.token;
  delete parsed.json;

  // Extract --fields filter
  const fieldsFilter = parsed.fields;
  delete parsed.fields;

  const noAuthCommands = ["list-commands", "get-token"];
  const needsAuth = !noAuthCommands.includes(commandName);
  if (commandName !== "list-commands" && !baseUrl) { console.error("Error: YOUTRACK_BASE_URL not set. Use --base-url or env var."); process.exit(1); }
  if (needsAuth && !token) { console.error("Error: YOUTRACK_TOKEN not set. Use --token or env var."); process.exit(1); }

  // Read-only mode: block write commands before running handler
  if (process.env.YOUTRACK_RO === "1" && isWriteCommand(commandName)) {
    console.error(JSON.stringify({ error: "Read-only mode active (YOUTRACK_RO=1). Write operations are blocked." }, null, 2));
    process.exit(1);
  }

  // Validate required params
  for (const p of command.params) {
    if (!p.startsWith("?") && !(p in parsed)) {
      console.error(`Missing required parameter: --${p}`);
      console.error(`Usage: youtrack-cli ${commandName} ${command.params.map(p => p.startsWith("?") ? `[--${p.slice(1)}]` : `--${p}`).join(" ")}`);
      process.exit(1);
    }
  }

  // Map optional params (strip ? prefix)
  const args = {};
  for (const p of command.params) {
    const name = p.startsWith("?") ? p.slice(1) : p;
    args[name] = parsed[name] || undefined;
  }

  try {
    const ctx = { base: baseUrl ? baseUrl.replace(/\/$/, "") : "", token: token || "" };
    let result = await command.handler(ctx, args);

    // Apply --fields filter
    if (fieldsFilter === "?") {
      const getKeys = (obj) => {
        if (Array.isArray(obj) && obj.length > 0) return getKeys(obj[0]);
        return obj && typeof obj === "object" ? Object.keys(obj) : [];
      };
      result = { availableFields: getKeys(result) };
    } else if (fieldsFilter) {
      result = filterFields(result, fieldsFilter.split(",").map(s => s.trim()));
    }

    if (!quiet) {
      if (typeof result === "string") process.stdout.write(result + "\n");
      else console.log(JSON.stringify(result, null, 2));
    }
  } catch (err) {
    const errOut = { error: err.message };
    if (err.suggestion) errOut.suggestion = err.suggestion;
    else if (COMMAND_SUGGESTIONS[commandName]) errOut.suggestion = COMMAND_SUGGESTIONS[commandName];
    console.error(JSON.stringify(errOut, null, 2));
    process.exit(1);
  }
}

main();
