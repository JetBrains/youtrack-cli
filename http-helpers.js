// HTTP helpers for YouTrack CLI — zero-dependency, Node.js built-in only.

import { request as httpsRequest } from "https";
import { request as httpRequest } from "http";
import { URL } from "url";
import { readFileSync, existsSync } from "fs";

// ── HTTP core ────────────────────────────────────────────────────────────────

export function httpReq(method, baseUrl, path, token, body = null) {
  return new Promise((resolve, reject) => {
    const url = new URL(path, baseUrl);
    const isHttps = url.protocol === "https:";
    const options = {
      hostname: url.hostname,
      port: url.port || (isHttps ? 443 : 80),
      path: url.pathname + url.search,
      method,
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${token}`,
      },
    };
    if (body !== null) {
      const data = JSON.stringify(body);
      options.headers["Content-Type"] = "application/json";
      options.headers["Content-Length"] = Buffer.byteLength(data);
    }
    const req = (isHttps ? httpsRequest : httpRequest)(options, (res) => {
      let chunks = [];
      res.on("data", (c) => chunks.push(c));
      res.on("end", () => {
        const text = Buffer.concat(chunks).toString();
        if (res.statusCode >= 300) {
          reject(new Error(`HTTP ${res.statusCode}: ${text}`));
        } else {
          try { resolve(text ? JSON.parse(text) : null); }
          catch { resolve(text); }
        }
      });
    });
    req.on("error", reject);
    if (body !== null) req.write(JSON.stringify(body));
    req.end();
  });
}

export function doGet(base, path, token) { return httpReq("GET", base, path, token); }
export function doPost(base, path, token, body) { return httpReq("POST", base, path, token, body); }
export function doPut(base, path, token, body) { return httpReq("PUT", base, path, token, body); }
export function doDelete(base, path, token) { return httpReq("DELETE", base, path, token); }

export function httpReqBasicAuth(method, baseUrl, path, login, password, body = null) {
  return new Promise((resolve, reject) => {
    const url = new URL(path, baseUrl);
    const isHttps = url.protocol === "https:";
    const credentials = Buffer.from(`${login}:${password}`).toString("base64");
    const options = {
      hostname: url.hostname,
      port: url.port || (isHttps ? 443 : 80),
      path: url.pathname + url.search,
      method,
      headers: {
        Accept: "application/json",
        Authorization: `Basic ${credentials}`,
      },
    };
    if (body !== null) {
      const data = JSON.stringify(body);
      options.headers["Content-Type"] = "application/json";
      options.headers["Content-Length"] = Buffer.byteLength(data);
    }
    const req = (isHttps ? httpsRequest : httpRequest)(options, (res) => {
      let chunks = [];
      res.on("data", (c) => chunks.push(c));
      res.on("end", () => {
        const text = Buffer.concat(chunks).toString();
        if (res.statusCode >= 300) {
          reject(new Error(`HTTP ${res.statusCode}: ${text}`));
        } else {
          try { resolve(text ? JSON.parse(text) : null); }
          catch { resolve(text); }
        }
      });
    });
    req.on("error", reject);
    if (body !== null) req.write(JSON.stringify(body));
    req.end();
  });
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
      const g = allGroups.find(g => g.name === n.trim());
      if (!g) throw new Error(`Group '${n.trim()}' not found`);
      return { id: g.id, name: g.name };
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

export function findByName(list, name, entityLabel) {
  const item = list.find(i => i.name === name);
  if (!item) throw new Error(`${entityLabel} '${name}' not found`);
  return item;
}

export function findByLogin(list, login, entityLabel) {
  const item = list.find(i => i.login === login);
  if (!item) throw new Error(`${entityLabel} '${login}' not found`);
  return item;
}

export function findByShortName(list, shortName, entityLabel) {
  const item = list.find(i => i.shortName === shortName);
  if (!item) throw new Error(`${entityLabel} '${shortName}' not found`);
  return item;
}
