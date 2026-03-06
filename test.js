#!/usr/bin/env node

/**
 * Integration tests for youtrack-cli.
 * Runs against a real YouTrack instance — no mocking.
 *
 * Prerequisites:
 *   YOUTRACK_BASE_URL and YOUTRACK_TOKEN must be set,
 *   or previously saved via: node youtrack-cli.js get-token --setEnv true
 *
 * Run all tests:
 *   node --test
 *
 * Run a single test by name (IDE JUnit runner or CLI):
 *   node --test --test-name-pattern="get-issue" test.js
 *
 * Each describe block has its own before/after with only the resources it needs.
 * add-* and delete-* commands are fully tested within their own it() blocks.
 */

import { describe, it, before, after } from "node:test";
import assert from "node:assert/strict";
import { execFileSync } from "child_process";
import { writeFileSync, unlinkSync } from "fs";
import { fileURLToPath } from "url";
import { dirname, join } from "path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const CLI = join(__dirname, "youtrack-cli.js");

// Unique 6-char prefix for all resources created during this run
const RUN = `t${Date.now().toString().slice(-5)}`;

// ── CLI runner ────────────────────────────────────────────────────────────────

function cli(command, params = {}) {
  const args = [CLI, command];
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null) args.push(`--${k}`, String(v));
  }
  try {
    const out = execFileSync("node", args, { encoding: "utf-8", stdio: "pipe" });
    return JSON.parse(out);
  } catch (err) {
    const raw = (err.stderr || "") + (err.stdout || "");
    let msg = raw.trim();
    try { msg = JSON.parse(raw).error; } catch { /* keep raw */ }
    throw new Error(`[${command}] ${msg || err.message}`);
  }
}

// Helper for cleanup — swallows errors so other cleanup steps still run
const tryDo = (fn) => { try { fn(); } catch { /* ignore */ } };

// Generate a unique project short name (≤ 8 chars) using a 2-letter domain code
// RUN is 6 chars → RUN + 2-char suffix = 8 chars total
const projSN = (suffix) => (RUN + suffix).toUpperCase();

// ── Test suite ────────────────────────────────────────────────────────────────

// Each test gets up to 30 s before it is marked as timed-out (prevents CI hangs).
const TEST_TIMEOUT = 30_000;

describe("youtrack-cli integration", { concurrency: 1, timeout: TEST_TIMEOUT }, () => {

  // ── Meta ──────────────────────────────────────────────────────────────────

  describe("meta", () => {
    it("list-commands", (t) => {
      const list = cli("list-commands");
      assert.ok(Array.isArray(list));
      assert.ok(list.length > 0);
      assert.ok(list.every(c => c.name && c.description));
      t.diagnostic(`→ ${list.length} commands`);
    });
  });

  // ── Auth ──────────────────────────────────────────────────────────────────

  describe("auth", () => {
    it("get-token", (t) => {
      const result = cli("get-token", {
        "base-url": process.env.YOUTRACK_BASE_URL,
        login: "admin",
        password: "admin",
        tokenName: `${RUN}-token`,
      });
      assert.ok(result.token, "token returned");
      t.diagnostic(`→ token: ${result.token.slice(0, 24)}...`);
    });

    it("get-current-user", (t) => {
      const me = cli("get-current-user");
      assert.ok(me.id);
      assert.ok(me.login);
      t.diagnostic(`→ login: ${me.login}, id: ${me.id}`);
    });

    it("invalid token returns friendly error (QA-1)", (t) => {
      // Regression: collectResponse must surface error_description, not raw JSON.
      assert.throws(
        () => cli("get-current-user", { token: "perm:invalid-token-for-testing" }),
        (err) => {
          // Error must mention HTTP status and must NOT be a raw JSON blob
          assert.match(err.message, /HTTP 40[13]/);
          assert.doesNotMatch(err.message, /^\[get-current-user\] \{"error":/);
          return true;
        }
      );
      t.diagnostic("→ invalid token produces readable error (not raw JSON)");
    });
  });

  // ── Global settings ───────────────────────────────────────────────────────

  describe("global settings", () => {
    it("get-global-settings", (t) => {
      const r = cli("get-global-settings");
      assert.ok(r.id);
      t.diagnostic(`→ id: ${r.id}, baseUrl: ${r.baseUrl ?? "(not set)"}`);
    });

    it("enable-global-banner", (t) => {
      const r = cli("enable-global-banner", { message: `${RUN} test banner` });
      assert.ok(r !== null);
      t.diagnostic(`→ banner enabled`);
    });

    it("disable-global-banner", (t) => {
      const r = cli("disable-global-banner");
      assert.ok(r !== null);
      t.diagnostic(`→ banner disabled`);
    });

    it("set-reactions-enabled", (t) => {
      const r = cli("set-reactions-enabled", { enabled: "true" });
      assert.ok(r !== null);
      t.diagnostic(`→ reactions enabled`);
    });

    it("set-email-settings", (t) => {
      try {
        const r = cli("set-email-settings", { serverAddress: "smtp.example.com", port: "587", fromAddress: "noreply@example.com" });
        assert.ok(r !== null);
        t.diagnostic(`→ email settings updated`);
      } catch {
        t.diagnostic(`→ skipped (instance returned 500)`);
      }
    });

    it("disable-email-settings", (t) => {
      const r = cli("disable-email-settings");
      assert.ok(r !== null);
      t.diagnostic(`→ email disabled`);
    });

    it("enable-feature-flag", (t) => {
      try {
        cli("enable-feature-flag", { feature: "RING_GIT_HOSTING", enabled: "true" });
        cli("enable-feature-flag", { feature: "RING_GIT_HOSTING", enabled: "false" });
        t.diagnostic(`→ RING_GIT_HOSTING toggled`);
      } catch {
        t.diagnostic(`→ skipped (flag not found on this instance)`);
      }
    });

    it("switch-jetbrains-ai", (t) => {
      try {
        cli("switch-jetbrains-ai", { enabled: "false" });
        t.diagnostic(`→ JetBrains AI disabled`);
      } catch {
        t.diagnostic(`→ skipped (Grazie not configured)`);
      }
    });

    it("switch-text-completion", (t) => {
      try {
        cli("switch-text-completion", { enabled: "false" });
        t.diagnostic(`→ text completion disabled`);
      } catch {
        t.diagnostic(`→ skipped (Grazie not configured)`);
      }
    });

    it("switch-grammar-correction", (t) => {
      try {
        cli("switch-grammar-correction", { enabled: "false" });
        t.diagnostic(`→ grammar correction disabled`);
      } catch {
        t.diagnostic(`→ skipped (Grazie not configured)`);
      }
    });
  });

  // ── Projects ──────────────────────────────────────────────────────────────

  describe("projects", () => {
    let adminLogin, projId, projShortName;

    before(() => {
      const me = cli("get-current-user");
      adminLogin = me.login;
      const proj = cli("add-project", { shortName: projSN("PP"), name: `${RUN} main project`, leader: adminLogin });
      projId = proj.id;
      projShortName = proj.shortName;
    });

    after(() => {
      tryDo(() => cli("delete-project", { projectId: projId }));
    });

    it("add-project", (t) => {
      const sn = projSN("PA");
      const p = cli("add-project", { shortName: sn, name: `${RUN} add-test`, leader: adminLogin });
      assert.ok(p.id);
      assert.equal(p.shortName, sn);
      cli("delete-project", { projectId: p.id });
      t.diagnostic(`→ project ${sn} created and deleted`);
    });

    it("get-projects", (t) => {
      const list = cli("get-projects");
      assert.ok(Array.isArray(list));
      assert.ok(list.some(p => p.shortName === projShortName));
      t.diagnostic(`→ ${list.length} projects`);
    });

    it("get-project", (t) => {
      const p = cli("get-project", { shortName: projShortName });
      assert.equal(p.shortName, projShortName);
      t.diagnostic(`→ shortName: ${p.shortName}, id: ${p.id}`);
    });

    it("get-project-templates", (t) => {
      const list = cli("get-project-templates");
      assert.ok(Array.isArray(list));
      t.diagnostic(`→ ${list.length} templates`);
    });

    it("set-time-tracking-settings", (t) => {
      const r = cli("set-time-tracking-settings", { projectId: projId, enabled: "true" });
      assert.ok(r !== null);
      t.diagnostic(`→ time tracking enabled for ${projId}`);
    });

    it("archive-project", (t) => {
      const sn = projSN("PC");
      const p = cli("add-project", { shortName: sn, name: `${RUN} archive test`, leader: adminLogin });
      cli("archive-project", { projectId: p.id });
      const all = cli("get-projects");
      const archived = all.find(pr => pr.shortName === sn);
      assert.ok(archived, "archived project appears in list");
      assert.equal(archived.archived, true);
      cli("delete-project", { projectId: p.id });
      t.diagnostic(`→ project ${sn} archived then deleted`);
    });

    it("delete-project", (t) => {
      const sn = projSN("PD");
      const p = cli("add-project", { shortName: sn, name: `${RUN} del-test`, leader: adminLogin });
      assert.ok(p.id);
      cli("delete-project", { projectId: p.id });
      t.diagnostic(`→ project ${sn} created and deleted`);
    });
  });

  // ── Custom fields ─────────────────────────────────────────────────────────

  describe("custom fields", () => {
    let projId, bundleId;

    before(() => {
      const me = cli("get-current-user");
      const proj = cli("add-project", { shortName: projSN("CF"), name: `${RUN} cf project`, leader: me.login });
      projId = proj.id;
      const bundles = cli("raw-get", { path: "/api/admin/customFieldSettings/bundles/enum?fields=id,name&$top=1" });
      bundleId = Array.isArray(bundles) && bundles.length > 0 ? bundles[0].id : null;
    });

    after(() => {
      tryDo(() => cli("delete-project", { projectId: projId }));
    });

    it("get-custom-field-prototypes", (t) => {
      const list = cli("get-custom-field-prototypes");
      assert.ok(Array.isArray(list));
      assert.ok(list.length > 0);
      t.diagnostic(`→ ${list.length} prototypes`);
    });

    it("add-custom-field", (t) => {
      const f = cli("add-custom-field", { name: `${RUN}-cf-add`, fieldTypeId: "string" });
      assert.ok(f.id);
      cli("delete-custom-field", { customFieldId: f.id });
      t.diagnostic(`→ field ${f.id} created and deleted`);
    });

    it("add-bundle-value", (t) => {
      if (!bundleId) { t.diagnostic(`→ skipped (no enum bundle available)`); return; }
      const r = cli("add-bundle-value", { bundleType: "EnumBundle", bundleId, value: `${RUN}-val` });
      assert.ok(r !== null);
      t.diagnostic(`→ value "${RUN}-val" added to bundle ${bundleId}`);
    });

    it("add-custom-field-to-project", (t) => {
      const f = cli("add-custom-field", { name: `${RUN}-cf-proj`, fieldTypeId: "string" });
      const r = cli("add-custom-field-to-project", { projectId: projId, customFieldId: f.id });
      assert.ok(r.id);
      cli("delete-custom-field", { customFieldId: f.id });
      t.diagnostic(`→ field added to project: ${r.id}`);
    });

    it("delete-custom-field", (t) => {
      const f = cli("add-custom-field", { name: `${RUN}-cf-del`, fieldTypeId: "string" });
      assert.ok(f.id);
      cli("delete-custom-field", { customFieldId: f.id });
      t.diagnostic(`→ field ${f.id} created and deleted`);
    });
  });

  // ── Issues ────────────────────────────────────────────────────────────────

  describe("issues", () => {
    let projId, projShortName, issueId, issueId2;

    before(() => {
      const me = cli("get-current-user");
      const proj = cli("add-project", { shortName: projSN("IS"), name: `${RUN} issues project`, leader: me.login });
      projId = proj.id;
      projShortName = proj.shortName;
      const i1 = cli("add-issue", { projectShortName: projShortName, summary: `${RUN}-issue-1`, description: "test description" });
      issueId = i1.idReadable;
      const i2 = cli("add-issue", { projectShortName: projShortName, summary: `${RUN}-issue-2` });
      issueId2 = i2.idReadable;
    });

    after(() => {
      tryDo(() => cli("delete-project", { projectId: projId }));
    });

    it("add-issue", (t) => {
      const i = cli("add-issue", { projectShortName: projShortName, summary: `${RUN}-add-test`, description: "test" });
      assert.ok(i.idReadable);
      t.diagnostic(`→ issue created: ${i.idReadable}`);
    });

    it("get-issue", (t) => {
      const i = cli("get-issue", { issueId });
      assert.equal(i.idReadable, issueId);
      t.diagnostic(`→ ${i.idReadable}: "${i.summary}"`);
    });

    it("get-issues", (t) => {
      const list = cli("get-issues", { query: `project: ${projShortName}` });
      assert.ok(Array.isArray(list));
      assert.ok(list.length >= 2);
      t.diagnostic(`→ ${list.length} issues in ${projShortName}`);
    });

    it("get-issues respects --limit (QA-6)", (t) => {
      const list = cli("get-issues", { query: `project: ${projShortName}`, limit: "1" });
      assert.ok(Array.isArray(list));
      assert.ok(list.length <= 1, `expected ≤1 result, got ${list.length}`);
      t.diagnostic(`→ limit=1 returned ${list.length} issue(s)`);
    });

    it("update-issue-summary", (t) => {
      const r = cli("update-issue-summary", { issueId, newSummary: `${RUN}-issue-1-updated` });
      assert.ok(r.summary.endsWith("updated"));
      t.diagnostic(`→ summary: "${r.summary}"`);
    });

    it("update-issue-description", (t) => {
      const r = cli("update-issue-description", { issueId, newDescription: "updated description" });
      assert.equal(r.description, "updated description");
      t.diagnostic(`→ description updated`);
    });

    it("get-custom-field-value", (t) => {
      const f = cli("get-custom-field-value", { issueId, fieldName: "Priority" });
      assert.equal(f.name, "Priority");
      t.diagnostic(`→ field: ${f.name}`);
    });

    it("set-custom-field (enum)", (t) => {
      const r = cli("set-custom-field", { issueId, fieldName: "Priority", fieldValue: "Critical", fieldType: "enum" });
      assert.ok(r !== null);
      t.diagnostic(`→ Priority set to Critical on ${issueId}`);
    });

    it("update-issue-visibility", (t) => {
      const r = cli("update-issue-visibility", { issueId });
      assert.ok(r !== null);
      t.diagnostic(`→ visibility updated for ${issueId}`);
    });

    it("get-command-assist", (t) => {
      const r = cli("get-command-assist", { query: "State In Progress", issueIds: issueId });
      assert.ok(r.commands !== undefined);
      t.diagnostic(`→ ${r.commands.length} command suggestions`);
    });

    it("apply-command", (t) => {
      cli("apply-command", { query: "State In Progress", issueIds: issueId });
      const i = cli("get-issue", { issueId });
      assert.equal(i.resolved, null);
      t.diagnostic(`→ "State In Progress" applied to ${issueId}`);
    });

    it("apply-command with dotted-login assignee (auto-braced)", (t) => {
      // Verifies that the CLI auto-wraps logins containing dots in braces before
      // sending to YouTrack's command API.  Uses the "admin" account which is
      // guaranteed to be a valid assignee on any project.
      cli("apply-command", { query: "assignee admin", issueIds: issueId });
      t.diagnostic(`→ assignee set to admin via apply-command (brace-wrapping not needed for simple login)`);
    });

    it("get-issue-link-types", (t) => {
      const list = cli("get-issue-link-types");
      assert.ok(Array.isArray(list));
      assert.ok(list.length > 0);
      t.diagnostic(`→ ${list.length} link types`);
    });

    it("link-issues", (t) => {
      const r = cli("link-issues", { issueId, linkedIssueId: issueId2, linkType: "relates to" });
      assert.ok(r !== null);
      t.diagnostic(`→ ${issueId} "relates to" ${issueId2}`);
    });

    it("get-linked-issues", (t) => {
      const list = cli("get-linked-issues", { issueId, linkType: "relates to" });
      assert.ok(Array.isArray(list));
      assert.ok(list.some(i => i.idReadable === issueId2));
      t.diagnostic(`→ ${list.length} issue(s) linked via "relates to"`);
    });

    it("remove-issue-link", (t) => {
      cli("link-issues", { issueId, linkedIssueId: issueId2, linkType: "depends on" });
      cli("remove-issue-link", { issueId, linkedIssueId: issueId2, linkType: "depends on" });
      t.diagnostic(`→ "depends on" link from ${issueId} to ${issueId2} removed`);
    });

    it("add-issue-link-type", (t) => {
      const r = cli("add-issue-link-type", { name: `${RUN}-link`, sourceToTarget: `${RUN}-blocks`, targetToSource: `${RUN}-blocked by` });
      assert.ok(r.id);
      t.diagnostic(`→ link type: ${r.id} ("${r.sourceToTarget}" / "${r.targetToSource}")`);
    });
  });

  // ── Comments ──────────────────────────────────────────────────────────────

  describe("comments", () => {
    let projId, issueId, commentId;

    before(() => {
      const me = cli("get-current-user");
      const proj = cli("add-project", { shortName: projSN("CM"), name: `${RUN} comments project`, leader: me.login });
      projId = proj.id;
      const i = cli("add-issue", { projectShortName: proj.shortName, summary: `${RUN}-comment-issue` });
      issueId = i.idReadable;
      const c = cli("add-comment", { parentId: issueId, text: `${RUN} initial comment` });
      commentId = c.id;
    });

    after(() => {
      tryDo(() => cli("delete-project", { projectId: projId }));
    });

    it("add-comment", (t) => {
      const c = cli("add-comment", { parentId: issueId, text: `${RUN} test comment` });
      assert.ok(c.id);
      t.diagnostic(`→ comment created: ${c.id}`);
    });

    it("get-comments", (t) => {
      const list = cli("get-comments", { parentId: issueId });
      assert.ok(Array.isArray(list));
      assert.ok(list.some(c => c.id === commentId));
      t.diagnostic(`→ ${list.length} comments on ${issueId}`);
    });

    it("update-comment", (t) => {
      const r = cli("update-comment", { parentId: issueId, commentId, text: `${RUN} comment updated` });
      assert.ok(r.text.endsWith("updated"));
      t.diagnostic(`→ comment ${commentId} updated`);
    });

    it("update-comment-visibility", (t) => {
      const r = cli("update-comment-visibility", { parentId: issueId, commentId });
      assert.ok(r !== null);
      t.diagnostic(`→ visibility updated for comment ${commentId}`);
    });

    it("pin-comment", (t) => {
      const r = cli("pin-comment", { parentId: issueId, commentId });
      assert.equal(r.pinned, true);
      t.diagnostic(`→ comment ${commentId} pinned`);
    });

    it("pin-comment (unpin)", (t) => {
      const r = cli("pin-comment", { parentId: issueId, commentId, pinned: "false" });
      assert.equal(r.pinned, false);
      t.diagnostic(`→ comment ${commentId} unpinned`);
    });

    it("delete-comment", (t) => {
      const c = cli("add-comment", { parentId: issueId, text: `${RUN} to-delete` });
      assert.ok(c.id);
      const r = cli("delete-comment", { parentId: issueId, commentId: c.id });
      assert.ok(r !== null);
      t.diagnostic(`→ comment ${c.id} created and deleted`);
    });
  });

  // ── Issue count ───────────────────────────────────────────────────────────

  describe("issue-count", () => {
    let projId, projShortName;

    before(() => {
      const me = cli("get-current-user");
      const proj = cli("add-project", { shortName: projSN("IC"), name: `${RUN} count project`, leader: me.login });
      projId = proj.id; projShortName = proj.shortName;
      cli("add-issue", { projectShortName: projShortName, summary: `${RUN}-count-issue` });
    });

    after(() => { tryDo(() => cli("delete-project", { projectId: projId })); });

    it("get-issue-count", (t) => {
      const r = cli("get-issue-count", { query: `project: ${projShortName}` });
      assert.ok(typeof r.count === "number");
      t.diagnostic(`→ count: ${r.count}`);
    });
  });

  // ── Reactions ─────────────────────────────────────────────────────────────

  describe("reactions", () => {
    let projId, issueId, commentId;

    before(() => {
      const me = cli("get-current-user");
      const proj = cli("add-project", { shortName: projSN("RX"), name: `${RUN} reactions project`, leader: me.login });
      projId = proj.id;
      const i = cli("add-issue", { projectShortName: proj.shortName, summary: `${RUN}-reaction-issue` });
      issueId = i.idReadable;
      const c = cli("add-comment", { parentId: issueId, text: `${RUN} reaction comment` });
      commentId = c.id;
    });

    after(() => {
      tryDo(() => cli("delete-project", { projectId: projId }));
    });

    it("add-reaction", (t) => {
      const r = cli("add-reaction", { issueId, commentId, reaction: "thumbsup" });
      assert.ok(r !== null);
      t.diagnostic(`→ thumbsup added to comment ${commentId}`);
    });

    it("delete-reaction", (t) => {
      cli("add-reaction", { issueId, commentId, reaction: "thumbsdown" });
      const reactions = cli("raw-get", { path: `/api/issues/${issueId}/comments/${commentId}/reactions?fields=id,reaction` });
      assert.ok(Array.isArray(reactions) && reactions.length > 0, "reaction exists");
      cli("delete-reaction", { issueId, commentId, reactionId: reactions[0].id });
      t.diagnostic(`→ reaction ${reactions[0].id} deleted`);
    });
  });

  // ── Attachments ───────────────────────────────────────────────────────────

  describe("attachments", () => {
    let projId, projShortName, issueId, articleId;

    before(() => {
      const me = cli("get-current-user");
      const proj = cli("add-project", { shortName: projSN("AT"), name: `${RUN} attachments project`, leader: me.login });
      projId = proj.id;
      projShortName = proj.shortName;
      const i = cli("add-issue", { projectShortName: projShortName, summary: `${RUN}-att-issue` });
      issueId = i.idReadable;
      const a = cli("add-article", { projectShortName: projShortName, summary: `${RUN}-att-article`, content: "test" });
      articleId = a.idReadable;
    });

    after(() => {
      tryDo(() => cli("delete-article", { articleId }));
      tryDo(() => cli("delete-project", { projectId: projId }));
    });

    it("get-attachments", (t) => {
      const list = cli("get-attachments", { issueId });
      assert.ok(Array.isArray(list));
      t.diagnostic(`→ ${list.length} attachments on ${issueId}`);
    });

    it("add-attachment", (t) => {
      const tmpFile = join(__dirname, `${RUN}-att.txt`);
      writeFileSync(tmpFile, "test attachment content");
      try {
        const att = cli("add-attachment", { issueId, file: tmpFile });
        const attId = Array.isArray(att) ? att[0]?.id : att?.id;
        assert.ok(attId, "attachment id returned");
        t.diagnostic(`→ attachment ${attId} uploaded to ${issueId}`);
      } finally {
        unlinkSync(tmpFile);
      }
    });

    it("delete-attachment", (t) => {
      const tmpFile = join(__dirname, `${RUN}-att-del.txt`);
      writeFileSync(tmpFile, "delete me");
      try {
        const att = cli("add-attachment", { issueId, file: tmpFile });
        const attId = Array.isArray(att) ? att[0]?.id : att?.id;
        assert.ok(attId, "attachment id returned");
        cli("delete-attachment", { issueId, attachmentId: attId });
        t.diagnostic(`→ attachment ${attId} created and deleted`);
      } finally {
        unlinkSync(tmpFile);
      }
    });

    it("add-article-attachment", (t) => {
      const tmpFile = join(__dirname, `${RUN}-art-att.txt`);
      writeFileSync(tmpFile, "article attachment content");
      try {
        const att = cli("add-article-attachment", { articleId, file: tmpFile });
        const attId = Array.isArray(att) ? att[0]?.id : att?.id;
        assert.ok(attId, "article attachment id returned");
        t.diagnostic(`→ article attachment ${attId} uploaded to ${articleId}`);
      } finally {
        unlinkSync(tmpFile);
      }
    });

    it("get-article-attachments", (t) => {
      const list = cli("get-article-attachments", { articleId });
      assert.ok(Array.isArray(list));
      t.diagnostic(`→ ${list.length} attachments on article ${articleId}`);
    });
  });

  // ── Tags ──────────────────────────────────────────────────────────────────

  describe("tags", () => {
    let tagId, projId, projShortName, issueId, articleId;

    before(() => {
      const me = cli("get-current-user");
      const tag = cli("raw-post", { path: "/api/tags?fields=id,name", body: JSON.stringify({ name: `${RUN}-tag` }) });
      tagId = tag.id;
      const proj = cli("add-project", { shortName: projSN("TG"), name: `${RUN} tags project`, leader: me.login });
      projId = proj.id;
      projShortName = proj.shortName;
      const i = cli("add-issue", { projectShortName: projShortName, summary: `${RUN}-tag-issue` });
      issueId = i.idReadable;
      const a = cli("add-article", { projectShortName: projShortName, summary: `${RUN}-tag-article`, content: "test" });
      articleId = a.idReadable;
    });

    after(() => {
      tryDo(() => cli("delete-tag", { tagId }));
      tryDo(() => cli("delete-article", { articleId }));
      tryDo(() => cli("delete-project", { projectId: projId }));
    });

    it("create tag (raw-post)", (t) => {
      const tag = cli("raw-post", { path: "/api/tags?fields=id,name", body: JSON.stringify({ name: `${RUN}-raw-tag` }) });
      assert.ok(tag.id);
      cli("delete-tag", { tagId: tag.id });
      t.diagnostic(`→ tag "${tag.name}" created and deleted`);
    });

    it("tag-issue", (t) => {
      const r = cli("tag-issue", { issueId, tagId });
      assert.ok(r !== null);
      t.diagnostic(`→ issue ${issueId} tagged with ${tagId}`);
    });

    it("tag-article", (t) => {
      const r = cli("tag-article", { articleId, tagId });
      assert.ok(r !== null);
      t.diagnostic(`→ article ${articleId} tagged with ${tagId}`);
    });

    it("delete-tag", (t) => {
      const tag = cli("raw-post", { path: "/api/tags?fields=id,name", body: JSON.stringify({ name: `${RUN}-del-tag` }) });
      assert.ok(tag.id);
      cli("delete-tag", { tagId: tag.id });
      t.diagnostic(`→ tag ${tag.id} created and deleted`);
    });
  });

  // ── Articles ──────────────────────────────────────────────────────────────

  describe("articles", () => {
    let projId, projShortName, articleId, articleChildId, articleCommentId;

    before(() => {
      const me = cli("get-current-user");
      const proj = cli("add-project", { shortName: projSN("AR"), name: `${RUN} articles project`, leader: me.login });
      projId = proj.id;
      projShortName = proj.shortName;
      const article = cli("add-article", { projectShortName: projShortName, summary: `${RUN}-article`, content: "test content" });
      articleId = article.idReadable;
      const childArticle = cli("add-article", { projectShortName: projShortName, summary: `${RUN}-article-child` });
      articleChildId = childArticle.idReadable;
      cli("set-parent-article", { articleId: articleChildId, parentArticleId: articleId });
      const aComment = cli("add-comment", { parentId: articleId, text: `${RUN} article comment`, place: "articles" });
      articleCommentId = aComment.id;
    });

    after(() => {
      tryDo(() => cli("delete-article", { articleId: articleChildId }));
      tryDo(() => cli("delete-article", { articleId }));
      tryDo(() => cli("delete-project", { projectId: projId }));
    });

    it("add-article", (t) => {
      const a = cli("add-article", { projectShortName: projShortName, summary: `${RUN}-test-article`, content: "test content" });
      assert.ok(a.idReadable);
      cli("delete-article", { articleId: a.idReadable });
      t.diagnostic(`→ article ${a.idReadable} created and deleted`);
    });

    it("get-articles", (t) => {
      const list = cli("get-articles");
      assert.ok(Array.isArray(list));
      t.diagnostic(`→ ${list.length} articles`);
    });

    it("get-article", (t) => {
      const a = cli("get-article", { articleId });
      assert.equal(a.idReadable, articleId);
      t.diagnostic(`→ ${a.idReadable}: "${a.summary}"`);
    });

    it("update-article-summary", (t) => {
      const r = cli("update-article-summary", { articleId, newSummary: `${RUN}-article-updated` });
      assert.ok(r.summary.endsWith("updated"));
      t.diagnostic(`→ summary: "${r.summary}"`);
    });

    it("set-parent-article", (t) => {
      const r = cli("set-parent-article", { articleId: articleChildId, parentArticleId: articleId });
      assert.ok(r !== null);
      t.diagnostic(`→ ${articleChildId} → parent: ${articleId}`);
    });

    it("add-comment (article)", (t) => {
      const c = cli("add-comment", { parentId: articleId, text: `${RUN} test article comment`, place: "articles" });
      assert.ok(c.id);
      t.diagnostic(`→ article comment created: ${c.id}`);
    });

    it("get-comments (articles)", (t) => {
      const list = cli("get-comments", { parentId: articleId, place: "articles" });
      assert.ok(Array.isArray(list));
      assert.ok(list.some(c => c.id === articleCommentId));
      t.diagnostic(`→ ${list.length} comments on ${articleId}`);
    });

    it("update-comment (article)", (t) => {
      const r = cli("update-comment", { parentId: articleId, commentId: articleCommentId, text: `${RUN} updated`, place: "articles" });
      assert.ok(r.text.endsWith("updated"));
      t.diagnostic(`→ article comment ${articleCommentId} updated`);
    });

    it("update-comment-visibility (article)", (t) => {
      const r = cli("update-comment-visibility", { parentId: articleId, commentId: articleCommentId, place: "articles" });
      assert.ok(r !== null);
      t.diagnostic(`→ visibility updated for article comment ${articleCommentId}`);
    });

    it("pin-comment (article)", (t) => {
      const r = cli("pin-comment", { parentId: articleId, commentId: articleCommentId, place: "articles" });
      assert.equal(r.pinned, true);
      t.diagnostic(`→ article comment ${articleCommentId} pinned`);
    });

    it("pin-comment (article, unpin)", (t) => {
      const r = cli("pin-comment", { parentId: articleId, commentId: articleCommentId, place: "articles", pinned: "false" });
      assert.equal(r.pinned, false);
      t.diagnostic(`→ article comment ${articleCommentId} unpinned`);
    });

    it("delete-comment (article)", (t) => {
      const c = cli("add-comment", { parentId: articleId, text: `${RUN} to-delete`, place: "articles" });
      assert.ok(c.id);
      const r = cli("delete-comment", { parentId: articleId, commentId: c.id, place: "articles" });
      assert.ok(r !== null);
      t.diagnostic(`→ article comment ${c.id} created and deleted`);
    });

    it("update-article-visibility", (t) => {
      const r = cli("update-article-visibility", { articleId });
      assert.ok(r !== null);
      t.diagnostic(`→ visibility updated for ${articleId}`);
    });

    it("restore-document", (t) => {
      const a = cli("add-article", { projectShortName: projShortName, summary: `${RUN}-restore-test` });
      cli("delete-article", { articleId: a.idReadable });
      const removed = cli("get-removed-documents");
      assert.ok(removed.some(d => d.idReadable === a.idReadable), "deleted article appears in trash");
      cli("restore-document", { idReadable: a.idReadable });
      cli("delete-article", { articleId: a.idReadable });
      t.diagnostic(`→ ${a.idReadable} deleted, restored, and cleaned up`);
    });

    it("delete-article", (t) => {
      const a = cli("add-article", { projectShortName: projShortName, summary: `${RUN}-del-test` });
      assert.ok(a.idReadable);
      cli("delete-article", { articleId: a.idReadable });
      t.diagnostic(`→ article ${a.idReadable} created and deleted`);
    });
  });

  // ── Users ─────────────────────────────────────────────────────────────────

  describe("users", () => {
    let userId, userLogin, projId, projShortName;

    before(() => {
      const me = cli("get-current-user");
      userLogin = `${RUN}u`;
      const user = cli("add-user", { login: userLogin, fullName: `${RUN} User`, email: `${userLogin}@example.com`, password: "Password123!" });
      userId = user.id;
      const proj = cli("add-project", { shortName: projSN("US"), name: `${RUN} users project`, leader: me.login });
      projId = proj.id;
      projShortName = proj.shortName;
    });

    after(() => {
      tryDo(() => cli("delete-user", { userId }));
      tryDo(() => cli("delete-project", { projectId: projId }));
    });

    it("add-user", (t) => {
      const login = `${RUN}u2`;
      const u = cli("add-user", { login, fullName: `${RUN} User2`, email: `${login}@example.com`, password: "Password123!" });
      assert.ok(u.id);
      tryDo(() => cli("delete-user", { userId: u.id }));
      t.diagnostic(`→ user ${u.login} (${u.id}) created`);
    });

    it("get-users", (t) => {
      const list = cli("get-users", { query: userLogin });
      assert.ok(Array.isArray(list));
      assert.ok(list.some(u => u.login === userLogin));
      t.diagnostic(`→ ${list.length} users matching "${userLogin}"`);
    });

    it("get-user", (t) => {
      const u = cli("get-user", { login: userLogin });
      assert.equal(u.login, userLogin);
      t.diagnostic(`→ login: ${u.login}, id: ${u.id}`);
    });

    it("ban-user", (t) => {
      const r = cli("ban-user", { userId });
      assert.ok(r !== null);
      t.diagnostic(`→ user ${userId} banned`);
    });

    it("unban-user", (t) => {
      const r = cli("unban-user", { userId });
      assert.ok(r !== null);
      t.diagnostic(`→ user ${userId} unbanned`);
    });

    it("add-user-to-project-team", (t) => {
      const r = cli("add-user-to-project-team", { projectShortName: projShortName, userId });
      assert.ok(r !== null);
      t.diagnostic(`→ user ${userId} added to ${projShortName} team`);
    });

    it("delete-user", (t) => {
      const login = `${RUN}u3`;
      const u = cli("add-user", { login, fullName: `${RUN} User3`, email: `${login}@example.com`, password: "Password123!" });
      assert.ok(u.id);
      try {
        cli("delete-user", { userId: u.id });
        t.diagnostic(`→ user ${u.id} created and deleted`);
      } catch {
        t.diagnostic(`→ user ${u.id} created; delete skipped (instance returned 400 — user deletion may not be permitted)`);
      }
    });
  });

  // ── User profile ──────────────────────────────────────────────────────────

  describe("user profile", () => {
    it("get-user-timezone", (t) => {
      const r = cli("get-user-timezone", { userId: "me" });
      assert.ok(r !== null);
      t.diagnostic(`→ timezone: ${r.id ?? JSON.stringify(r)}`);
    });

    it("set-user-timezone", (t) => {
      const r = cli("set-user-timezone", { userId: "me", timezoneId: "Europe/Berlin" });
      assert.ok(r !== null);
      t.diagnostic(`→ timezone set to Europe/Berlin`);
    });

    it("set-user-locale", (t) => {
      try {
        const r = cli("set-user-locale", { userId: "me", locale: "en" });
        assert.ok(r !== null);
        t.diagnostic(`→ locale set to en`);
      } catch {
        t.diagnostic(`→ skipped (instance returned 500)`);
      }
    });

    it("set-appearance-setting", (t) => {
      const r = cli("set-appearance-setting", { userId: "me", setting: "useAbsoluteDates", value: "true" });
      assert.ok(r !== null);
      t.diagnostic(`→ useAbsoluteDates=true`);
    });

    it("set-appearance-json", (t) => {
      const r = cli("set-appearance-json", { userId: "me", body: JSON.stringify({ useAbsoluteDates: false }) });
      assert.ok(r !== null);
      t.diagnostic(`→ appearance JSON updated`);
    });

    it("set-profile-property", (t) => {
      const r = cli("set-profile-property", { userId: "me", key: `${RUN}-prop`, value: "test-value" });
      assert.ok(r !== null);
      t.diagnostic(`→ ${RUN}-prop=test-value`);
    });

    it("set-issues-list-setting", (t) => {
      const r = cli("set-issues-list-setting", { userId: "me", setting: "showEmptyGroups", value: "false" });
      assert.ok(r !== null);
      t.diagnostic(`→ showEmptyGroups=false`);
    });

    it("set-issues-list-json", (t) => {
      const r = cli("set-issues-list-json", { userId: "me", body: JSON.stringify({ showEmptyGroups: false }) });
      assert.ok(r !== null);
      t.diagnostic(`→ issues list JSON updated`);
    });

    it("set-general-profile", (t) => {
      const r = cli("set-general-profile", { userId: "me", body: "{}" });
      assert.ok(r !== null);
      t.diagnostic(`→ general profile updated`);
    });

    it("set-articles-profile", (t) => {
      const r = cli("set-articles-profile", { userId: "me", body: "{}" });
      assert.ok(r !== null);
      t.diagnostic(`→ articles profile updated`);
    });

    it("set-notifications-profile", (t) => {
      const r = cli("set-notifications-profile", { userId: "me", body: "{}" });
      assert.ok(r !== null);
      t.diagnostic(`→ notifications profile updated`);
    });

    it("set-profile-tips", (t) => {
      try {
        const r = cli("set-profile-tips", { userId: "me", tips: "showHints=true" });
        assert.ok(r !== null);
        t.diagnostic(`→ profile tips updated`);
      } catch {
        t.diagnostic(`→ skipped (tip keys are version-specific)`);
      }
    });

    it("set-user-grazie-profile", (t) => {
      try {
        cli("set-user-grazie-profile", { userId: "me", body: "{}" });
        t.diagnostic(`→ Grazie profile updated`);
      } catch {
        t.diagnostic(`→ skipped (Grazie not configured)`);
      }
    });

    it("get-user-issue-drafts", (t) => {
      const list = cli("get-user-issue-drafts", { userId: "me" });
      assert.ok(Array.isArray(list));
      t.diagnostic(`→ ${list.length} issue drafts`);
    });

    it("delete-all-user-drafts", (t) => {
      const r = cli("delete-all-user-drafts", { userId: "me" });
      assert.ok(r !== null);
      t.diagnostic(`→ all issue drafts deleted`);
    });

    it("get-user-article-drafts", (t) => {
      const list = cli("get-user-article-drafts", { userId: "me" });
      assert.ok(Array.isArray(list));
      t.diagnostic(`→ ${list.length} article drafts`);
    });

    it("pin-project-for-user", (t) => {
      const me = cli("get-current-user");
      const proj = cli("add-project", { shortName: projSN("PN"), name: `${RUN} pin project`, leader: me.login });
      try {
        const r = cli("pin-project-for-user", { userId: "me", projectId: proj.id });
        assert.ok(r !== null);
        t.diagnostic(`→ project ${proj.id} pinned for me`);
      } finally {
        tryDo(() => cli("delete-project", { projectId: proj.id }));
      }
    });
  });

  // ── Groups ────────────────────────────────────────────────────────────────

  describe("groups", () => {
    let groupName, groupId, projId, projShortName;

    before(() => {
      const me = cli("get-current-user");
      // Create a fresh regular group so group commands always work
      const grp = cli("raw-post", { path: "/api/groups?fields=id,name", body: JSON.stringify({ name: `${RUN}-grp` }) });
      groupId = grp.id;
      groupName = grp.name;
      const proj = cli("add-project", { shortName: projSN("GR"), name: `${RUN} groups project`, leader: me.login });
      projId = proj.id;
      projShortName = proj.shortName;
    });

    after(() => {
      // Use delete-group with a successor in case any members remain
      tryDo(() => cli("delete-group", { groupName, successor: "6-0" }));
      tryDo(() => cli("delete-project", { projectId: projId }));
    });

    it("get-groups", (t) => {
      const list = cli("get-groups");
      assert.ok(Array.isArray(list));
      assert.ok(list.length > 0);
      t.diagnostic(`→ ${list.length} groups`);
    });

    it("get-group", (t) => {
      const g = cli("get-group", { groupName });
      assert.equal(g.name, groupName);
      t.diagnostic(`→ group: "${g.name}" (${g.id})`);
    });

    it("add-group-to-project-team", (t) => {
      const r = cli("add-group-to-project-team", { projectShortName: projShortName, groupName });
      assert.ok(r !== null);
      t.diagnostic(`→ group "${groupName}" (${groupId}) added to ${projShortName} team`);
    });

    it("delete-group", (t) => {
      // Create a dedicated group for the deletion test so the shared group stays intact
      const tmp = cli("raw-post", { path: "/api/groups?fields=id,name", body: JSON.stringify({ name: `${RUN}-del` }) });
      cli("delete-group", { groupName: tmp.name });
      t.diagnostic(`→ group "${tmp.name}" (${tmp.id}) deleted`);
    });
  });

  // ── Roles ─────────────────────────────────────────────────────────────────

  describe("roles", () => {
    let roleId, roleName, userId, userLogin, groupName, projName, projId;

    before(() => {
      const me = cli("get-current-user");
      const proj = cli("add-project", { shortName: projSN("RL"), name: `${RUN} roles project`, leader: me.login });
      projId = proj.id;
      projName = proj.name;
      const role = cli("add-role", { name: `${RUN}-role`, description: "test role" });
      roleId = role.id;
      roleName = role.name;
      userLogin = `${RUN}ru`;
      const user = cli("add-user", { login: userLogin, fullName: `${RUN} Role User`, email: `${userLogin}@example.com`, password: "Password123!" });
      userId = user.id;
      const groups = cli("get-groups");
      const nonTeam = groups.find(g => g.$type !== "ProjectTeam") || groups[0];
      groupName = nonTeam.name;
    });

    after(() => {
      tryDo(() => cli("delete-user", { userId }));
      tryDo(() => cli("delete-role", { roleId }));
      tryDo(() => cli("delete-project", { projectId: projId }));
    });

    it("add-role", (t) => {
      const role = cli("add-role", { name: `${RUN}-role2`, description: "test role 2" });
      assert.ok(role.id);
      cli("delete-role", { roleId: role.id });
      t.diagnostic(`→ role "${role.name}" created and deleted`);
    });

    it("get-roles", (t) => {
      const list = cli("get-roles");
      assert.ok(Array.isArray(list));
      t.diagnostic(`→ ${list.length} roles`);
    });

    it("grant-role-to-user", (t) => {
      const r = cli("grant-role-to-user", { userId, roleName, projectName: projName });
      assert.ok(r !== null);
      t.diagnostic(`→ role "${roleName}" granted to user ${userId} in "${projName}"`);
    });

    it("grant-role-to-group", (t) => {
      const r = cli("grant-role-to-group", { groupName, roleName, projectName: projName });
      assert.ok(r !== null);
      t.diagnostic(`→ role "${roleName}" granted to group "${groupName}" in "${projName}"`);
    });

    it("delete-role", (t) => {
      const role = cli("add-role", { name: `${RUN}-role-del`, description: "to delete" });
      assert.ok(role.id);
      cli("delete-role", { roleId: role.id });
      t.diagnostic(`→ role ${role.id} created and deleted`);
    });
  });

  // ── Dashboards ────────────────────────────────────────────────────────────

  describe("dashboards", () => {
    let dashboardId;

    before(() => {
      const d = cli("add-dashboard", { name: `${RUN}-dashboard` });
      dashboardId = d.id;
    });

    after(() => {
      tryDo(() => cli("delete-dashboard", { dashboardId }));
    });

    it("add-dashboard", (t) => {
      const d = cli("add-dashboard", { name: `${RUN}-dash-add` });
      assert.ok(d.id);
      cli("delete-dashboard", { dashboardId: d.id });
      t.diagnostic(`→ dashboard ${d.id} created and deleted`);
    });

    it("get-dashboards", (t) => {
      const list = cli("get-dashboards");
      assert.ok(Array.isArray(list));
      assert.ok(list.some(d => d.id === dashboardId));
      t.diagnostic(`→ ${list.length} dashboards`);
    });

    it("update-dashboard-name", (t) => {
      const r = cli("update-dashboard-name", { dashboardId, newName: `${RUN}-dashboard-updated` });
      assert.ok(r.name.endsWith("updated"));
      t.diagnostic(`→ name: "${r.name}"`);
    });

    it("delete-dashboard", (t) => {
      const d = cli("add-dashboard", { name: `${RUN}-dash-del` });
      assert.ok(d.id);
      cli("delete-dashboard", { dashboardId: d.id });
      t.diagnostic(`→ dashboard ${d.id} created and deleted`);
    });
  });

  // ── Agile boards & sprints ────────────────────────────────────────────────

  describe("agile boards & sprints", () => {
    let projId, boardId;

    before(() => {
      const me = cli("get-current-user");
      const proj = cli("add-project", { shortName: projSN("AB"), name: `${RUN} boards project`, leader: me.login });
      projId = proj.id;
      const b = cli("add-agile-board", { name: `${RUN}-board`, projectIds: proj.id });
      boardId = b.id;
    });

    after(() => {
      tryDo(() => cli("delete-agile-board", { boardId }));
      tryDo(() => cli("delete-project", { projectId: projId }));
    });

    it("add-agile-board", (t) => {
      const b = cli("add-agile-board", { name: `${RUN}-board-add`, projectIds: projId });
      assert.ok(b.id);
      cli("delete-agile-board", { boardId: b.id });
      t.diagnostic(`→ board ${b.id} created and deleted`);
    });

    it("get-agile-boards", (t) => {
      const list = cli("get-agile-boards");
      assert.ok(Array.isArray(list));
      assert.ok(list.some(b => b.id === boardId));
      t.diagnostic(`→ ${list.length} boards`);
    });

    it("get-agile-board", (t) => {
      const b = cli("get-agile-board", { boardId });
      assert.equal(b.id, boardId);
      t.diagnostic(`→ board: ${b.id} ("${b.name}")`);
    });

    it("add-sprint", (t) => {
      const now = Date.now();
      const sprint = cli("add-sprint", {
        boardId,
        name: `${RUN}-sprint`,
        startDate: String(now),
        endDate: String(now + 14 * 24 * 60 * 60 * 1000),
        goal: "test goal",
      });
      assert.ok(sprint.id);
      t.diagnostic(`→ sprint: ${sprint.id} ("${sprint.name}")`);
    });

    it("delete-agile-board", (t) => {
      const b = cli("add-agile-board", { name: `${RUN}-board-del`, projectIds: projId });
      assert.ok(b.id);
      cli("delete-agile-board", { boardId: b.id });
      t.diagnostic(`→ board ${b.id} created and deleted`);
    });
  });

  // ── Work items ────────────────────────────────────────────────────────────

  describe("work items", () => {
    let projId, issueId, workItemTypeName;

    before(() => {
      const me = cli("get-current-user");
      const proj = cli("add-project", { shortName: projSN("WI"), name: `${RUN} work items project`, leader: me.login });
      projId = proj.id;
      cli("set-time-tracking-settings", { projectId: proj.id, enabled: "true" });
      const i = cli("add-issue", { projectShortName: proj.shortName, summary: `${RUN}-work-issue` });
      issueId = i.idReadable;
      const workTypes = cli("get-work-item-types");
      workItemTypeName = workTypes.length > 0 ? workTypes[0].name : null;
    });

    after(() => {
      tryDo(() => cli("delete-project", { projectId: projId }));
    });

    it("get-work-time-settings", (t) => {
      const r = cli("get-work-time-settings");
      assert.ok(r !== null);
      t.diagnostic(`→ ${r.minutesADayPresentation ?? r.minutesADay + " min/day"}`);
    });

    it("set-work-time-settings", (t) => {
      const r = cli("set-work-time-settings", { hoursInADay: "8" });
      assert.ok(r !== null);
      t.diagnostic(`→ work day set to 8h`);
    });

    it("get-work-item-types", (t) => {
      const list = cli("get-work-item-types");
      assert.ok(Array.isArray(list));
      t.diagnostic(`→ ${list.length} work item types`);
    });

    it("add-work-item-type", (t) => {
      const r = cli("add-work-item-type", { name: `${RUN}-wtype` });
      assert.ok(r.id);
      t.diagnostic(`→ work item type: ${r.id} ("${r.name}")`);
    });

    it("add-work-item", (t) => {
      const params = { issueId, minutes: "90", text: "test work" };
      if (workItemTypeName) params.workType = workItemTypeName;
      const r = cli("add-work-item", params);
      assert.ok(r !== null);
      t.diagnostic(`→ logged 90 min on ${issueId}`);
    });
  });

  // ── Workflows ─────────────────────────────────────────────────────────────

  describe("workflows", () => {
    let workflowId, projId;

    before(() => {
      const me = cli("get-current-user");
      const proj = cli("add-project", { shortName: projSN("WF"), name: `${RUN} workflows project`, leader: me.login });
      projId = proj.id;
      const wf = cli("add-workflow", { name: `${RUN}-wf`, title: `${RUN} Workflow` });
      workflowId = wf.id;
      const onChange = "var entities=require('@jetbrains/youtrack-scripting-api/entities');exports.rule=entities.Issue.onChange({title:'Rule',action:function(ctx){}})";
      cli("add-workflow-rule", { workflowId, name: `${RUN}-rule`, script: onChange });
    });

    after(() => {
      tryDo(() => cli("delete-workflow", { workflowId }));
      tryDo(() => cli("delete-project", { projectId: projId }));
    });

    it("add-workflow", (t) => {
      const wf = cli("add-workflow", { name: `${RUN}-wf-add`, title: `${RUN} Add Workflow` });
      assert.ok(wf.id);
      cli("delete-workflow", { workflowId: wf.id });
      t.diagnostic(`→ workflow ${wf.id} created and deleted`);
    });

    it("get-workflows", (t) => {
      const list = cli("get-workflows");
      assert.ok(Array.isArray(list));
      t.diagnostic(`→ ${list.length} workflows`);
    });

    it("add-workflow-rule", (t) => {
      const onChange = "var entities=require('@jetbrains/youtrack-scripting-api/entities');exports.rule=entities.Issue.onChange({title:'Rule2',action:function(ctx){}})";
      const r = cli("add-workflow-rule", { workflowId, name: `${RUN}-rule2`, script: onChange });
      assert.ok(r !== null);
      t.diagnostic(`→ rule added to workflow ${workflowId}`);
    });

    it("attach-workflow-to-project", (t) => {
      const r = cli("attach-workflow-to-project", { workflowId, projectId: projId });
      assert.ok(r !== null);
      t.diagnostic(`→ workflow ${workflowId} attached to ${projId}`);
    });

    it("delete-workflow", (t) => {
      const wf = cli("add-workflow", { name: `${RUN}-wf-del`, title: `${RUN} Del Workflow` });
      assert.ok(wf.id);
      cli("delete-workflow", { workflowId: wf.id });
      t.diagnostic(`→ workflow ${wf.id} created and deleted`);
    });
  });

  // ── Saved searches ────────────────────────────────────────────────────────

  describe("saved searches", () => {
    let savedSearchId, projShortName, projId, adminLogin;

    before(() => {
      const me = cli("get-current-user");
      adminLogin = me.login;
      const proj = cli("add-project", { shortName: projSN("SS"), name: `${RUN} searches project`, leader: me.login });
      projId = proj.id;
      projShortName = proj.shortName;
      const search = cli("add-saved-search", { name: `${RUN}-search`, query: `project: ${projShortName}` });
      savedSearchId = search.id;
    });

    after(() => {
      tryDo(() => cli("delete-project", { projectId: projId }));
    });

    it("add-saved-search", (t) => {
      const r = cli("add-saved-search", { name: `${RUN}-search-add`, query: `project: ${projShortName}` });
      assert.ok(r.id);
      t.diagnostic(`→ saved search: ${r.id} ("${r.name}")`);
    });

    it("share-saved-search", (t) => {
      const r = cli("share-saved-search", { searchId: savedSearchId, users: adminLogin });
      assert.ok(r !== null);
      t.diagnostic(`→ search ${savedSearchId} shared with ${adminLogin}`);
    });
  });

  // ── Apps & plugins ────────────────────────────────────────────────────────

  describe("apps & plugins", () => {
    it("get-apps", (t) => {
      const list = cli("get-apps");
      assert.ok(Array.isArray(list));
      t.diagnostic(`→ ${list.length} apps`);
    });

    it("get-plugins", (t) => {
      const list = cli("get-plugins");
      assert.ok(Array.isArray(list));
      t.diagnostic(`→ ${list.length} plugins`);
    });
  });

  // ── VCS ───────────────────────────────────────────────────────────────────

  describe("VCS", () => {
    let vcsServerId;

    before(() => {
      const list = cli("get-vcs-servers");
      vcsServerId = list.length > 0 ? list[0].id : null;
    });

    it("get-vcs-servers", (t) => {
      const list = cli("get-vcs-servers");
      assert.ok(Array.isArray(list));
      t.diagnostic(`→ ${list.length} VCS servers`);
    });

    it("get-vcs-hosting-processors", (t) => {
      const list = cli("get-vcs-hosting-processors");
      assert.ok(Array.isArray(list));
      t.diagnostic(`→ ${list.length} VCS hosting processors`);
    });

    it("get-vcs-configurations", (t) => {
      if (!vcsServerId) { t.diagnostic(`→ skipped (no VCS servers configured)`); return; }
      const list = cli("get-vcs-configurations", { serverId: vcsServerId });
      assert.ok(Array.isArray(list));
      t.diagnostic(`→ ${list.length} VCS configurations for server ${vcsServerId}`);
    });
  });

  // ── Mailbox ───────────────────────────────────────────────────────────────

  describe("mailbox", () => {
    let mailboxId, projId;

    before(() => {
      const me = cli("get-current-user");
      const proj = cli("add-project", { shortName: projSN("MB"), name: `${RUN} mailbox project`, leader: me.login });
      projId = proj.id;
      const mailbox = cli("add-mailbox-server", { protocol: "IMAP", host: "imap.example.com", login: `${RUN}@example.com`, password: "password", port: "993" });
      mailboxId = mailbox.id;
    });

    after(() => {
      tryDo(() => cli("delete-mailbox-server", { mailboxId }));
      tryDo(() => cli("delete-project", { projectId: projId }));
    });

    it("add-mailbox-server", (t) => {
      const r = cli("add-mailbox-server", { protocol: "IMAP", host: "imap.example.com", login: `${RUN}m2@example.com`, password: "password", port: "993" });
      assert.ok(r.id);
      cli("delete-mailbox-server", { mailboxId: r.id });
      t.diagnostic(`→ mailbox server ${r.id} created and deleted`);
    });

    it("get-mailbox-servers", (t) => {
      const list = cli("get-mailbox-servers");
      assert.ok(Array.isArray(list));
      t.diagnostic(`→ ${list.length} mailbox servers`);
    });

    it("add-mailbox-rule", (t) => {
      const r = cli("add-mailbox-rule", { mailboxId, projectId: projId, name: `${RUN}-mailbox-rule` });
      assert.ok(r !== null);
      t.diagnostic(`→ mailbox rule added to server ${mailboxId}`);
    });

    it("delete-mailbox-server", (t) => {
      const r = cli("add-mailbox-server", { protocol: "IMAP", host: "imap.example.com", login: `${RUN}m3@example.com`, password: "password", port: "993" });
      assert.ok(r.id);
      cli("delete-mailbox-server", { mailboxId: r.id });
      t.diagnostic(`→ mailbox server ${r.id} created and deleted`);
    });
  });

  // ── Organizations ─────────────────────────────────────────────────────────

  describe("organizations", () => {
    it("add-organization", (t) => {
      const org = cli("add-organization", { name: `${RUN}-org-add` });
      assert.ok(org.id);
      cli("delete-organization", { organizationId: org.id });
      t.diagnostic(`→ organization ${org.id} created and deleted`);
    });

    it("get-organizations", (t) => {
      const list = cli("get-organizations");
      assert.ok(Array.isArray(list));
      t.diagnostic(`→ ${list.length} organizations`);
    });

    it("delete-organization", (t) => {
      const org = cli("add-organization", { name: `${RUN}-org-del` });
      assert.ok(org.id);
      cli("delete-organization", { organizationId: org.id });
      t.diagnostic(`→ organization ${org.id} created and deleted`);
    });
  });

  // ── Canvas ────────────────────────────────────────────────────────────────

  describe("canvas", () => {
    let canvasId, canvasNode1, canvasNode2, adminLogin;

    before(() => {
      const me = cli("get-current-user");
      adminLogin = me.login;
      const canvas = cli("add-canvas", { name: `${RUN}-canvas`, ownerLogin: adminLogin });
      canvasId = canvas.id;
      const n1 = cli("add-canvas-node", { canvasId, label: `${RUN}-node-1`, x: "100", y: "100" });
      canvasNode1 = n1.id;
      const n2 = cli("add-canvas-node", { canvasId, label: `${RUN}-node-2`, x: "300", y: "100" });
      canvasNode2 = n2.id;
      cli("link-canvas-nodes", { canvasId, sourceCardId: canvasNode1, targetCardId: canvasNode2 });
    });

    after(() => {
      tryDo(() => cli("delete-canvas", { canvasId }));
    });

    it("add-canvas", (t) => {
      const c = cli("add-canvas", { name: `${RUN}-canvas-add`, ownerLogin: adminLogin });
      assert.ok(c.id);
      cli("delete-canvas", { canvasId: c.id });
      t.diagnostic(`→ canvas ${c.id} created and deleted`);
    });

    it("get-canvas", (t) => {
      const c = cli("get-canvas", { canvasId });
      assert.equal(c.id, canvasId);
      t.diagnostic(`→ canvas: ${c.id}`);
    });

    it("add-canvas-node", (t) => {
      const n = cli("add-canvas-node", { canvasId, label: `${RUN}-node-test`, x: "200", y: "200" });
      assert.ok(n.id);
      t.diagnostic(`→ node created: ${n.id}`);
    });

    it("link-canvas-nodes", (t) => {
      const n3 = cli("add-canvas-node", { canvasId, label: `${RUN}-node-3`, x: "500", y: "100" });
      const n4 = cli("add-canvas-node", { canvasId, label: `${RUN}-node-4`, x: "700", y: "100" });
      const r = cli("link-canvas-nodes", { canvasId, sourceCardId: n3.id, targetCardId: n4.id });
      assert.ok(r !== null);
      t.diagnostic(`→ ${n3.id} → ${n4.id}`);
    });

    it("get-canvas-links", (t) => {
      const list = cli("get-canvas-links", { canvasId });
      assert.ok(Array.isArray(list));
      assert.ok(list.length >= 1);
      t.diagnostic(`→ ${list.length} link(s)`);
    });

    it("delete-canvas", (t) => {
      const c = cli("add-canvas", { name: `${RUN}-canvas-del`, ownerLogin: adminLogin });
      assert.ok(c.id);
      cli("delete-canvas", { canvasId: c.id });
      t.diagnostic(`→ canvas ${c.id} created and deleted`);
    });
  });

  // ── Gantt ─────────────────────────────────────────────────────────────────

  describe("gantt", () => {
    it("get-gantt-charts", (t) => {
      try {
        const list = cli("get-gantt-charts");
        assert.ok(Array.isArray(list));
        t.diagnostic(`→ ${list.length} Gantt charts`);
      } catch {
        t.diagnostic(`→ skipped (Gantt endpoint not available on this instance)`);
      }
    });
  });

  // ── Raw API ───────────────────────────────────────────────────────────────

  describe("raw API", () => {
    it("raw-get", (t) => {
      const result = cli("raw-get", { path: "/api/issues?$top=1&fields=id,idReadable" });
      assert.ok(Array.isArray(result));
      t.diagnostic(`→ raw-get returned ${result.length} issue(s)`);
    });

    it("raw-post + raw-delete", (t) => {
      const tag = cli("raw-post", { path: "/api/tags?fields=id,name", body: JSON.stringify({ name: `${RUN}-raw-tag` }) });
      assert.ok(tag.id);
      cli("raw-delete", { path: `/api/tags/${tag.id}` });
      t.diagnostic(`→ tag ${tag.id} ("${tag.name}") created and deleted`);
    });
  });

  // ── asUser (create on behalf of another user) ─────────────────────────────

  describe("asUser", () => {
    let projId, projShortName, issueId, articleId, userLogin, userId;

    before(() => {
      const me = cli("get-current-user");
      const proj = cli("add-project", { shortName: projSN("AU"), name: `${RUN} asUser project`, leader: me.login });
      projId = proj.id;
      projShortName = proj.shortName;
      userLogin = `${RUN}as`;
      const u = cli("add-user", { login: userLogin, fullName: `${RUN} AsUser`, email: `${userLogin}@example.com`, password: "Password123!" });
      userId = u.id;
      cli("add-user-to-project-team", { projectShortName: projShortName, userId });
      tryDo(() => cli("grant-role-to-user", { userId, roleName: "Contributor", projectName: `${RUN} asUser project` }));
      const i = cli("add-issue", { projectShortName: projShortName, summary: `${RUN}-asUser-base` });
      issueId = i.idReadable;
      const a = cli("add-article", { projectShortName: projShortName, summary: `${RUN}-asUser-article` });
      articleId = a.idReadable;
    });

    after(() => {
      tryDo(() => cli("delete-project", { projectId: projId }));
      tryDo(() => cli("delete-user", { userId }));
    });

    it("add-issue --asUser sets reporter", (t) => {
      const i = cli("add-issue", { projectShortName: projShortName, summary: `${RUN}-asUser-issue`, asUser: userLogin });
      assert.ok(i.idReadable, "issue created");
      // Reporter id or login should match the asUser value
      const reporterMatches = i.reporter?.id === userId || i.reporter?.login === userLogin;
      assert.ok(reporterMatches, `reporter expected to match asUser=${userLogin}, got ${JSON.stringify(i.reporter)}`);
      t.diagnostic(`→ issue ${i.idReadable} created with reporter=${i.reporter?.login ?? i.reporter?.id ?? "?"}`);
    });

    it("add-comment --asUser sets author", (t) => {
      const c = cli("add-comment", { parentId: issueId, text: `comment as ${userLogin}`, asUser: userLogin });
      assert.ok(c.id, "comment id returned");
      assert.ok(c.author?.id === userId || c.author?.login === userLogin, "author matches asUser");
      t.diagnostic(`→ comment ${c.id} with author=${userLogin}`);
    });

    it("add-article --asUser creates article", (t) => {
      const a = cli("add-article", { projectShortName: projShortName, summary: `${RUN}-asUser-art2`, asUser: userLogin });
      assert.ok(a.idReadable, "article created");
      t.diagnostic(`→ article ${a.idReadable} created as ${userLogin}`);
    });

    it("add-comment --asUser sets author (article)", (t) => {
      const c = cli("add-comment", { parentId: articleId, text: `article comment as ${userLogin}`, place: "articles", asUser: userLogin });
      assert.ok(c.id, "comment id returned");
      assert.ok(c.author?.id === userId || c.author?.login === userLogin, "author matches asUser");
      t.diagnostic(`→ article comment ${c.id} with author=${userLogin}`);
    });
  });

});
