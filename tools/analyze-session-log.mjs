// Analyze a DSH session jsonl.zstd log for seq corruption.
// Usage: node analyze-session-log.mjs <path-to-session.jsonl.zstd>
import { readFile } from "node:fs/promises";
import { zstdDecompress } from "node:zlib";
import { promisify } from "node:util";

const decompress = promisify(zstdDecompress);
const path = process.argv[2];
if (!path) { console.error("usage: node analyze-session-log.mjs <session.jsonl.zstd>"); process.exit(1); }

const buffer = await readFile(path);
console.log(`file size: ${buffer.length} bytes`);

// Structural split into zstd frames (concatenated-frame container).
const magic = Buffer.from([0x28, 0xb5, 0x2f, 0xfd]);
const frames = [];
let offset = 0;
while (offset < buffer.length) {
  if (!buffer.subarray(offset, offset + 4).equals(magic)) {
    console.error(`no frame magic at offset ${offset}; treating as torn tail start`);
    break;
  }
  const rest = buffer.subarray(offset + 4);
  const next = rest.indexOf(magic);
  const end = next === -1 ? buffer.length : offset + 4 + next;
  frames.push({ start: offset, end });
  offset = end;
}
console.log(`structural frames: ${frames.length}`);

const parts = [];
let tornStart = buffer.length;
let failedFrames = 0;
for (const f of frames) {
  try {
    parts.push(await decompress(buffer.subarray(f.start, f.end)));
    tornStart = f.end;
  } catch (e) {
    console.error(`frame ${f.start}..${f.end} decompress failed: ${e.message}`);
    failedFrames += 1;
    break;
  }
}
console.log(`failed/torn frames: ${failedFrames}, tornStart=${tornStart}`);
const plain = Buffer.concat(parts).toString("utf8");
const lines = plain.split("\n");
console.log(`plaintext: ${plain.length} bytes, ${lines.length} lines (incl. trailing empty)`);

const header = JSON.parse(lines[0]);
console.log(`header: type=${header.type} version=${header.version} id=${header.id} cwd=${header.cwd} createdAt=${new Date(header.createdAt).toISOString()}`);

function decodeRecord(value) {
  if (typeof value !== "object" || value === null) return [value];
  const tag = value.type;
  if (tag !== "text-chunks" && tag !== "reasoning-chunks" && tag !== "tool-call-chunks") return [value];
  const members = tag === "tool-call-chunks" ? value.data.args : value.data.texts;
  const events = [];
  let time = value.time0;
  for (let k = 0; k < members.length; k++) {
    if (k > 0) time += value.data.dt[k - 1];
    let chunk;
    if (tag === "text-chunks") chunk = { type: "text-delta", index: value.data.index, text: members[k] };
    else if (tag === "reasoning-chunks") chunk = { type: "reasoning-delta", index: value.data.index, text: members[k] };
    else chunk = { type: "tool-call-delta", index: value.data.index, id: value.data.id, ...(Object.hasOwn(value.data, "name") ? { name: value.data.name } : {}), argumentsDelta: members[k] };
    events.push({ type: "assistant/chunk", seq: value.seq0 + k, time, data: { turn: value.data.turn, step: value.data.step, chunk } });
  }
  return events;
}

const rowCount = lines.filter(Boolean).length - 1;
console.log(`event rows: ${rowCount}`);

// Pass 1: replicate SessionLogScanner exactly, find first issue and throw.
{
  let eventLine = 0;
  let events = [];
  let committedBytes = Buffer.byteLength(lines[0]) + 1;
  let issue = undefined;
  let firstIssueLine = undefined;
  let issueEvent = undefined;
  let throwLine = undefined;

  for (let li = 1; li < lines.length && throwLine === undefined; li++) {
    const line = lines[li];
    if (line.length === 0) continue;
    eventLine += 1;
    const endByte = committedBytes + Buffer.byteLength(line) + 1;
    let decoded;
    try { decoded = decodeRecord(JSON.parse(line)); }
    catch { issue ??= new Error(`unparsable at row ${eventLine}`); firstIssueLine ??= eventLine; continue; }
    if (issue !== undefined) { if (decoded.some((e) => e.type === "turn/end")) { throwLine = eventLine; } continue; }
    const rowStart = events.length;
    for (const event of decoded) {
      if (event.seq !== events.length) {
        issue = `expected ${events.length}, got ${event.seq}`;
        firstIssueLine ??= eventLine;
        issueEvent ??= event;
        events.length = rowStart;
        if (decoded.some((e) => e.type === "turn/end")) { throwLine = eventLine; }
        break;
      }
      events.push(event);
    }
    if (issue !== undefined && throwLine === undefined) { /* row skipped; committedBytes unchanged */ }
    else committedBytes = endByte;
  }

  console.log(`\n=== scanner replication ===`);
  console.log(`rows scanned: ${eventLine}`);
  console.log(`events accumulated before issue: ${events.length} (last good seq ${events.length - 1})`);
  console.log(`issue: ${issue}`);
  console.log(`first issue row: ${firstIssueLine}`);
  console.log(`throw row (contains turn/end): ${throwLine}`);
  console.log(`committedBytes (plaintext offset of last good row end): ${committedBytes}`);
  if (issueEvent) console.log(`issue event: ${JSON.stringify(issueEvent).slice(0, 500)}`);

  // Dump raw rows around the first issue row.
  if (firstIssueLine !== undefined) {
    const start = Math.max(1, firstIssueLine - 4);
    const end = Math.min(rowCount + 1, firstIssueLine + 4);
    console.log(`\n--- raw rows ${start}..${end} (1-based event row index) ---`);
    for (let r = start; r <= end; r++) {
      const line = lines[r];
      let summary = line.length > 260 ? line.slice(0, 260) + "…" : line;
      console.log(`row ${r}: ${summary}`);
    }
  }
}

// Pass 2: classify every row: running counter vs real seqs, find bounded region.
{
  let events = [];
  let row = 0;
  const badRows = [];
  let firstBad = undefined;
  let lastBad = undefined;
  for (let li = 1; li < lines.length; li++) {
    const line = lines[li];
    if (line.length === 0) continue;
    row += 1;
    let decoded;
    try { decoded = decodeRecord(JSON.parse(line)); } catch { continue; }
    const expected = events.length;
    let bad = false;
    for (const e of decoded) {
      if (e.seq !== events.length) { bad = true; break; }
      events.push(e);
    }
    if (bad) {
      lastBad = row;
      firstBad ??= row;
      badRows.push({
        row,
        expected,
        realSeq0: decodeRecord(JSON.parse(line))[0].seq ?? JSON.parse(line).seq0 ?? "?",
        types: decoded.map((e) => e.type),
        hasTurnEnd: decoded.some((e) => e.type === "turn/end"),
        turnEndCount: decoded.filter((e) => e.type === "turn/end").length,
      });
    }
  }
  console.log(`\n=== region classification ===`);
  console.log(`first bad row: ${firstBad}, last bad row: ${lastBad}, total bad rows: ${badRows.length}`);
  console.log(`rows after last bad row: ${rowCount - (lastBad ?? 0)}`);
  console.log(`\nfirst 12 bad rows:`);
  for (const b of badRows.slice(0, 12)) console.log(JSON.stringify(b).slice(0, 300));
  if (badRows.length > 12) {
    console.log(`\nlast 6 bad rows:`);
    for (const b of badRows.slice(-6)) console.log(JSON.stringify(b).slice(0, 300));
  }
  // count how many events in "good tail" after last bad row
  console.log(`\ntotal events accumulated with append-always: ${events.length} (last pushed seq would be ${events.length - 1})`);
}