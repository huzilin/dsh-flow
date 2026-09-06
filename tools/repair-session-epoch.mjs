// Repair DSH session logs with epoch-numbering corruption (backward or
// interleaved seq jumps after splice/end-seed seams).
//
// Usage:
//   node repair-session-epoch.mjs drop  <path> <firstRow> <lastRow>
//       Remove event rows firstRow..lastRow (1-based), keep the rest verbatim.
//       Validates the result is a contiguous 0..N scan before writing.
//   node repair-session-epoch.mjs compact <path>
//       Keep every row; split the event stream into maximal contiguous-seq
//       blocks in physical order; renumber each block sequentially so the
//       result is contiguous 0..N-1. Preserves all content and chronological
//       (physical) order.
//
// Both write headerFrame + bodyFrame (checksummed zstd) exactly like the
// backend's encoder, verify the candidate, and replace the file in place.
import { readFile, rename, stat, writeFile } from "node:fs/promises";
import { zstdCompress, zstdDecompress, constants } from "node:zlib";
import { promisify } from "node:util";

const compress = promisify(zstdCompress);
const decompress = promisify(zstdDecompress);
const CHECKSUM_OPTIONS = { params: { [constants.ZSTD_c_checksumFlag]: 1 } };

const mode = process.argv[2];
const path = process.argv[3];
if (!["drop", "compact"].includes(mode) || !path) {
  console.error("usage: node repair-session-epoch.mjs drop <path> <firstRow> <lastRow> | compact <path>");
  process.exit(1);
}

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
const plain = Buffer.concat(parts).toString("utf8");
const lines = plain.split("\n");
if (lines[lines.length - 1] === "") lines.pop();
const headerLine = lines[0];
console.log(`${path}\n  original: ${buffer.length} B, ${lines.length - 1} event rows`);

function decodeRecord(value) {
  if (typeof value !== "object" || value === null) return [value];
  const tag = value.type;
  if (tag !== "text-chunks" && tag !== "reasoning-chunks" && tag !== "tool-call-chunks") return [value];
  const members = tag === "tool-call-chunks" ? value.data.args : value.data.texts;
  const events = [];
  let time = value.time0;
  for (let k = 0; k < members.length; k++) {
    if (k > 0) time += value.data.dt[k - 1];
    events.push({ seq: value.seq0 + k, time });
  }
  return events;
}
function rowSeqRange(r) {
  const evs = decodeRecord(JSON.parse(lines[r]));
  return { first: evs[0].seq, last: evs[evs.length - 1].seq, n: evs.length };
}

// ---- Build the candidate row list + renumbering map ----
const kept = []; // { line, shift }
const verify = (rows) => {
  let seq = 0;
  let rowsScanned = 0;
  for (const row of rows) {
    rowsScanned += 1;
    let decoded;
    try { decoded = decodeRecord(JSON.parse(row.line)); }
    catch { throw new Error(`unparsable row ${rowsScanned}`); }
    for (const e of decoded) {
      const real = e.seq + row.shift;
      if (real !== seq) throw new Error(`seq gap at row ${rowsScanned}: expected ${seq}, got ${real}`);
      seq += 1;
    }
  }
  return seq;
};

if (mode === "drop") {
  const first = parseInt(process.argv[4], 10);
  const last = parseInt(process.argv[5], 10);
  console.log(`  drop rows ${first}..${last}`);
  if (!(first > 1 && last >= first && last < lines.length)) { console.error("ABORT: invalid row range"); process.exit(1); }
  // boundary sanity: row before drop ends with seq first-2; row after drop starts with seq last+1-? (native continuity)
  const before = rowSeqRange(first - 1);
  const after = rowSeqRange(last + 1);
  console.log(`  boundary: row ${first - 1} -> seq ${before.first}..${before.last}; row ${last + 1} -> seq ${after.first}..${after.last}`);
  if (before.last + 1 !== after.first) { console.error("ABORT: rows after the dropped region do not continue the rows before it"); process.exit(1); }
  for (let r = 1; r < lines.length; r++) {
    if (r >= first && r <= last) continue;
    kept.push({ line: lines[r], shift: 0 });
  }
} else {
  // compact: maximal contiguous blocks in physical order
  let blocks = [];
  let cur = null;
  let prevLast = null;
  for (let r = 1; r < lines.length; r++) {
    const { first: f, last: l } = rowSeqRange(r);
    if (cur === null || f !== prevLast + 1) { cur = { rows: [], firstSeq: f, lastSeq: l }; blocks.push(cur); }
    else cur.lastSeq = l;
    cur.rows.push(r);
    prevLast = l;
  }
  console.log(`  blocks: ${blocks.length}`);
  for (const b of blocks) console.log(`    rows ${b.rows[0]}..${b.rows[b.rows.length - 1]} seqs ${b.firstSeq}..${b.lastSeq} (${b.lastSeq - b.firstSeq + 1} events)`);
  let counter = 0;
  for (const b of blocks) {
    const shift = counter - b.firstSeq;
    counter += b.lastSeq - b.firstSeq + 1;
    for (const r of b.rows) kept.push({ line: lines[r], shift });
  }
}

const total = verify(kept);
console.log(`  verified: ${kept.length} rows, ${total} events, contiguous 0..${total - 1}`);

// ---- Encode & write ----
const headerFrame = await compress(Buffer.from(headerLine + "\n"), CHECKSUM_OPTIONS);
const body = kept.map((k) => {
  if (k.shift === 0) return k.line;
  // renumber event seqs in the raw line
  const v = JSON.parse(k.line);
  const tag = v.type;
  if (tag === "text-chunks" || tag === "reasoning-chunks" || tag === "tool-call-chunks") {
    v.seq0 = v.seq0 + k.shift;
  } else if (typeof v.seq === "number") {
    v.seq = v.seq + k.shift;
  } else {
    throw new Error(`unexpected row shape at renumber: ${k.line.slice(0, 80)}`);
  }
  return JSON.stringify(v);
}).join("\n") + "\n";
const bodyFrame = await compress(Buffer.from(body), CHECKSUM_OPTIONS);
const repaired = Buffer.concat([headerFrame, bodyFrame]);

const tmp = path + ".repaired";
await writeFile(tmp, repaired, { mode: 0o600 });
console.log(`  wrote ${repaired.length} B -> ${tmp}`);
// sanity: first frame is exactly one header line
{
  const b = await readFile(tmp);
  const idx = b.indexOf(Buffer.from([0x28, 0xb5, 0x2f, 0xfd]), 4);
  const head = await decompress(b.subarray(0, idx));
  if (!(head.indexOf(10) === head.length - 1 && head.includes(headerLine))) { console.error("ABORT: header frame invalid"); process.exit(1); }
}
await rename(tmp, path);
console.log(`  REPLACED ${path} (${(await stat(path)).size} B)`);