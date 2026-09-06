// Repair a DSH session jsonl.zstd whose committed region contains a duplicated
// replay segment (seq jump backward in the middle of the file).
//
// Usage: node repair-session-log.mjs <path-to-session.jsonl.zstd>
//
// Strategy (verified against the specific corruption):
//   file layout = [A: seqs 0..2729549 (turns 0..282)] [B: seqs 2598121..2612039
//   (re-run of turn 253; duplicate seq range)] [C: seqs 2729550..2731335
//   (turn 283, canonical continuation after agent/inbox/spliced seam)].
//   A + C is the contiguous authoritative timeline; B is dropped.
//
// The script asserts the boundary rows, rebuilds plaintext, verifies the
// rebuilt plaintext scans cleanly (monotonic seqs, no issue), re-encodes as
// headerFrame + bodyFrame (checksummed zstd, matching the backend layout),
// writes it to <path>.repaired, and only then replaces the original.
import { readFile, rename, writeFile, stat } from "node:fs/promises";
import { zstdCompress, zstdDecompress, constants } from "node:zlib";
import { promisify } from "node:util";

const compress = promisify(zstdCompress);
const decompress = promisify(zstdDecompress);
const CHECKSUM_OPTIONS = { params: { [constants.ZSTD_c_checksumFlag]: 1 } };

const path = process.argv[2];
if (!path) { console.error("usage: node repair-session-log.mjs <session.jsonl.zstd>"); process.exit(1); }

// ---- CLI overrides for the segment boundaries ----
// Split rows are 1-based indexes of event rows (line index in `lines` array).
const SPLIT_B_START = parseInt(process.argv[3] ?? "118819", 10); // first row of duplicate segment B
const SPLIT_B_END = parseInt(process.argv[4] ?? "120586", 10);   // last row of duplicate segment B

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
if (lines[lines.length - 1] === "") lines.pop(); // drop trailing empty line

console.log(`original: ${buffer.length} bytes, ${frames.length} frames, plaintext ${plain.length} B, ${lines.length - 1} event rows`);

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
function rowInfo(r) {
  const line = lines[r];
  let v, first, last;
  try { v = JSON.parse(line); } catch { return { row: r, unparsable: line.slice(0, 80) }; }
  const evs = decodeRecord(v);
  first = evs[0]; last = evs[evs.length - 1];
  return { row: r, type: v.type, firstSeq: first?.seq, lastSeq: last?.seq, n: evs.length, turn: v.data?.turn ?? evs[0]?.data?.turn, time: first?.time };
}

// ---- Assert the expected boundary rows so we never splice at a wrong spot ----
const aLast = rowInfo(SPLIT_B_START - 1); // last row of A
const bFirst = rowInfo(SPLIT_B_START);    // first row of B
const bLast = rowInfo(SPLIT_B_END);       // last row of B
const cFirst = rowInfo(SPLIT_B_END + 1);  // first row of C
const head = rowInfo(1);
console.log("boundary checks:");
console.log("  A head:  ", JSON.stringify(head));
console.log("  A last:  ", JSON.stringify(aLast));
console.log("  B first: ", JSON.stringify(bFirst));
console.log("  B last:  ", JSON.stringify(bLast));
console.log("  C first: ", JSON.stringify(cFirst));

// Explicit expectations:
const expectA = { turn: 282, lastSeq: 2729549, n: 1, type: "turn/end" };
const expectBLast = { turn: 253, lastSeq: 2612039, n: 1, type: "turn/end" };
const expectC = { type: "agent/inbox/spliced", firstSeq: 2729550, n: 1 };
const checks = [
  aLast.type === "turn/end" && aLast.turn === expectA.turn && aLast.lastSeq === expectA.lastSeq && aLast.n === 1,
  bFirst.lastSeq === 2598121 && bFirst.turn === 253,
  bLast.type === "turn/end" && bLast.turn === expectBLast.turn && bLast.lastSeq === expectBLast.lastSeq && bLast.n === 1,
  cFirst.type === "agent/inbox/spliced" && cFirst.firstSeq === expectC.firstSeq && cFirst.n === 1,
  aLast.lastSeq + 1 === cFirst.firstSeq, // C resumes seamlessly after A
  bFirst.firstSeq < aLast.lastSeq,        // B is a backward jump (duplicate/replay)
];
console.log("  checks:", checks);
if (!checks.every(Boolean)) {
  console.error("ABORT: boundary assertions failed — refusing to repair.");
  process.exit(1);
}

// ---- Rebuild candidate plaintext: header + A + C ----
const headerLine = lines[0];
const rowsA = lines.slice(1, SPLIT_B_START);          // rows 1..SPLIT_B_START-1
const rowsC = lines.slice(SPLIT_B_END + 1);           // rows SPLIT_B_END+1..end
console.log(`kept: header + ${rowsA.length} A rows + ${rowsC.length} C rows = ${rowsA.length + rowsC.length} event rows`);
const candidate = [headerLine, ...rowsA, ...rowsC].join("\n") + "\n";

// ---- Verify candidate scans cleanly (exact scanner semantics) ----
{
  const clines = candidate.split("\n");
  let events = 0;
  let issue;
  let scannedRows = 0;
  for (let li = 1; li < clines.length; li++) {
    const line = clines[li];
    if (line.length === 0) continue;
    scannedRows += 1;
    let decoded;
    try { decoded = decodeRecord(JSON.parse(line)); } catch { issue = `unparsable at row ${scannedRows}`; break; }
    for (const e of decoded) {
      if (e.seq !== events) { issue = `seq gap at ${scannedRows}: expected ${events}, got ${e.seq}`; break; }
      events += 1;
    }
    if (issue) break;
  }
  console.log(`candidate scan: ${scannedRows} rows, ${events} events (last seq ${events - 1}), issue: ${issue ?? "none"}`);
  if (issue || events - 1 !== 2731335) { console.error("ABORT: candidate does not verify."); process.exit(1); }
}

// ---- Encode (headerFrame + bodyFrame, checksummed) and write ----
const headerFrame = await compress(Buffer.from(headerLine + "\n"), CHECKSUM_OPTIONS);
const bodyFrame = await compress(Buffer.from([...rowsA, ...rowsC].join("\n") + "\n"), CHECKSUM_OPTIONS);
const repaired = Buffer.concat([headerFrame, bodyFrame]);

const tmp = path + ".repaired";
await writeFile(tmp, repaired, { mode: 0o600 });
console.log(`repaired temp written: ${repaired.length} bytes -> ${tmp}`);

// Decode the temp back and sanity-check the first frame is exactly one header line.
{
  const magic2 = Buffer.from([0x28, 0xb5, 0x2f, 0xfd]);
  const b = await readFile(tmp);
  const idx = b.indexOf(magic2, 4);
  const head = await decompress(b.subarray(0, idx));
  console.log("first frame plaintext:", JSON.stringify(head.toString()), "one-line:", head.indexOf(10) === head.length - 1);
}

await rename(tmp, path);
console.log(`REPLACED ${path}`);
const st = await stat(path);
console.log(`new size: ${st.size} bytes`);