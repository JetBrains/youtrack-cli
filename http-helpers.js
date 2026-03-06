// HTTP helpers for YouTrack CLI — zero-dependency, Node.js built-in only.

import { request as httpsRequest } from "https";
import { request as httpRequest } from "http";
import { URL } from "url";
import { readFileSync, existsSync } from "fs";

// ── HTTP core ────────────────────────────────────────────────────────────────

function collectResponse(res, resolve, reject) {
  const chunks = [];
  res.on("data", (c) => chunks.push(c));
  res.on("end", () => {
    const text = Buffer.concat(chunks).toString();
    if (res.statusCode >= 300) {
      let errMsg = `HTTP ${res.statusCode}`;
      try {
        const body = JSON.parse(text);
        const detail = body.error_description || body.error_developer_message || body.error || body.message;
        errMsg += detail ? `: ${detail}` : `: ${text}`;
      } catch {
        errMsg += text ? `: ${text}` : "";
      }
      if (res.statusCode === 401) errMsg += " — verify YOUTRACK_TOKEN is valid and not expired";
      if (res.statusCode === 403) errMsg += " — insufficient permissions for this operation";
      if (res.statusCode === 404) errMsg += " — resource not found; check the ID or short name";
      reject(new Error(errMsg));
    } else {
      try { resolve(text ? JSON.parse(text) : null); }
      catch { resolve(text); }
    }
  });
}

function buildUrl(baseUrl, path) {
  const base = new URL(baseUrl);
  const basePath = base.pathname.replace(/\/$/, "");
  return new URL(basePath + path, base.origin);
}

function httpReqCore(method, baseUrl, path, authHeader, body = null) {
  return new Promise((resolve, reject) => {
    const url = buildUrl(baseUrl, path);
    const isHttps = url.protocol === "https:";
    const data = body !== null ? JSON.stringify(body) : null;
    const headers = {
      Accept: "application/json",
      Authorization: authHeader,
    };
    if (data !== null) {
      headers["Content-Type"] = "application/json";
      headers["Content-Length"] = Buffer.byteLength(data);
    }
    const options = {
      hostname: url.hostname,
      port: url.port || (isHttps ? 443 : 80),
      path: url.pathname + url.search,
      method,
      headers,
    };
    const req = (isHttps ? httpsRequest : httpRequest)(options, (res) =>
      collectResponse(res, resolve, reject)
    );
    req.on("error", reject);
    if (data !== null) req.write(data);
    req.end();
  });
}

export function httpReq(method, baseUrl, path, token, body = null) {
  return httpReqCore(method, baseUrl, path, `Bearer ${token}`, body);
}

export function doGet(base, path, token) { return httpReq("GET", base, path, token); }
export function doPost(base, path, token, body) { return httpReq("POST", base, path, token, body); }
export function doPut(base, path, token, body) { return httpReq("PUT", base, path, token, body); }
export function doDelete(base, path, token) { return httpReq("DELETE", base, path, token); }

export function doMultipart(baseUrl, path, token, filePath, fileName, mimeType) {
  return new Promise((resolve, reject) => {
    const url = buildUrl(baseUrl, path);
    const isHttps = url.protocol === "https:";
    const fileData = readFileSync(filePath);
    const boundary = `----FormBoundary${Date.now().toString(16)}`;
    const header = Buffer.from(
      `--${boundary}\r\nContent-Disposition: form-data; name="attachment"; filename="${fileName}"\r\nContent-Type: ${mimeType}\r\n\r\n`
    );
    const footer = Buffer.from(`\r\n--${boundary}--\r\n`);
    const body = Buffer.concat([header, fileData, footer]);
    const options = {
      hostname: url.hostname,
      port: url.port || (isHttps ? 443 : 80),
      path: url.pathname + url.search,
      method: "POST",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${token}`,
        "Content-Type": `multipart/form-data; boundary=${boundary}`,
        "Content-Length": body.length,
      },
    };
    const req = (isHttps ? httpsRequest : httpRequest)(options, (res) =>
      collectResponse(res, resolve, reject)
    );
    req.on("error", reject);
    req.write(body);
    req.end();
  });
}

export function httpReqBasicAuth(method, baseUrl, path, login, password, body = null) {
  const credentials = Buffer.from(`${login}:${password}`).toString("base64");
  return httpReqCore(method, baseUrl, path, `Basic ${credentials}`, body);
}

// ── Utility helpers ──────────────────────────────────────────────────────────

export function enc(s) { return encodeURIComponent(s); }

export function resolveContent(value, filePath) {
  if (filePath) {
    if (!existsSync(filePath)) throw new Error(`File not found: ${filePath}`);
    return readFileSync(filePath, "utf-8");
  }
  return value || "";
}

export function parseVisibility(args, allGroups) {
  const body = { visibility: { $type: "LimitedVisibility" } };
  if (args.permittedGroups) {
    body.visibility.permittedGroups = args.permittedGroups.split(",").map(n => {
      const found = allGroups.find(g => g.name === n.trim());
      if (!found) throw new Error(`Group '${n.trim()}' not found`);
      return { id: found.id, name: found.name };
    });
  }
  if (args.permittedUsers) {
    body.visibility.permittedUsers = args.permittedUsers.split(",").map(l => ({ login: l.trim() }));
  }
  return body;
}

export function parseBool(value) {
  return value !== "false";
}

export function parseUserIdOrMe(userId) {
  return userId || "me";
}

export function parseIds(commaSeparated) {
  return commaSeparated.split(",").map(id => ({ id: id.trim() }));
}

function findBy(list, key, value, entityLabel) {
  const item = list.find(i => i[key] === value);
  if (!item) throw new Error(`${entityLabel} '${value}' not found`);
  return item;
}

export function findByName(list, name, entityLabel) { return findBy(list, "name", name, entityLabel); }
export function findByLogin(list, login, entityLabel) { return findBy(list, "login", login, entityLabel); }
export function findByShortName(list, shortName, entityLabel) { return findBy(list, "shortName", shortName, entityLabel); }
