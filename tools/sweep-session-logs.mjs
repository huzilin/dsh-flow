// Sweep all DSH session logs under a root for the seq-gap corruption pattern.
// Usage: node sweep-session-logs.mjs <sessions-root>
import { readdir, readFile, stat } from "node:fs/promises";
import { join } from "node:path";
import { zstdDecompress } from "node:zlib";
import { promisify } from "node:util";

const decompress = promisify(zstdDecompress);
const root = process.argv[2];
if (!root) { console.error("usage: node sweep-session-logs.mjs <sessions-root>"); process.exit(1); }

const logFiles = [];
async function walk(dir) {
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    const p = join(dir, entry.name);
    if (entry.isDirectory()) await walk(p);
    else if (entry.name.endsWith(".jsonl.zstd")) logFiles.push(p);
    else if (entry.name === "session.jsonl") logFiles.push(p);
  }
}
await walk(root);
logFiles.sort();
console.log(`found ${logFiles.length} session logs under ${root}\n`);

function decodeRecord(value) {
  if (typeof value !== "object" || value === null) return [value];
  const tag = value.type;
  if (tag !== "text-chunks" && tag !== "reasoning-chunks" && tag !== "tool-call-chunks") return [value];
  const members = tag === "tool-call-chunks" ? value.data.args : value.data.texts;
  const events = [];
  let time = value.time0;
  for (let k = 0; k < members.length; k++) {
    if (k > 0) time += value.data.dt[k - 1];
    events.push({ type: "assistant/chunk", seq: value.seq0 + k, time });
  }
  return events;
}

let corrupt = 0;
for (const path of logFiles) {
  const st = await stat(path);
  const id = path.split("/").pop();
  if (id.startsWith("session.jsonl.zstd.corrupt-backup")) continue;
  try {
    const buffer = await readFile(path);
    const magic = Buffer.from([0x28, 0xb5, 0x2f, 0xfd]);
    const frames = [];
    let offset = 0;
    while (offset < buffer.length) {
      if (!buffer.subarray(offset, offset + 4).equals(magic)) break;
      const rest = buffer.subarray(offset + 4);
      const next = rest.indexOf(magic);
      const end = next === -1 ? buffer.length : offset + 4 + next;
      frames.push({ start: offset, end });
      offset = end;
    }
    const parts = [];
    for (const f of frames) {
      try { parts.push(await decompress(buffer.subarray(f.start, f.end))); }
      catch { break; }
    }
    const lines = Buffer.concat(parts).toString("utf8").split("\n");
    // header line + rows
    let events = 0;
    let issue;
    let badRow = 0;
    let badSeq = "";
    for (let li = 1; li < lines.length; li++) {
      const line = lines[li];
      if (line.length === 0) continue;
      let decoded;
      try { decoded = decodeRecord(JSON.parse(line)); }
      catch { issue = "unparsable line"; badRow = li; break; }
      for (const e of decoded) {
        if (e.seq !== events) { issue = `seq gap (expected ${events}, got ${e.seq})`; badRow = li; badSeq = e.seq; break; }
        events += 1;
      }
      if (issue) break;
    }
    if (issue) {
      corrupt += 1;
      console.log(`CORRUPT  ${path}`);
      console.log(`         size=${st.size} issue=${issue} at event-row ${badRow} seq=${badSeq}`);
    } else {
      console.log(`ok       ${path}  (${st.size} B, ${events} events)`);
    }
  } catch (e) {
    console.log(`ERROR    ${path}: ${e.message}`);
  }
}
console.log(`\nsweep done: ${corrupt} corrupt log(s)`);