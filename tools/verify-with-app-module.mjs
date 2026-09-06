// Verify a DSH session log through the app's own persistence module.
// Usage: node verify-with-app-module.mjs <root> <session-id> <log-path>
const UNPACKED = "/Applications/DSH Desktop.app/Contents/Resources/app.asar.unpacked/node_modules";
const [root, sessionId, logPath] = process.argv.slice(2);
if (!root || !sessionId || !logPath) { console.error("usage: node verify-with-app-module.mjs <root> <session-id> <log-path>"); process.exit(1); }

const { default: JsonlSessionPersistence } = await import(`file://${UNPACKED}/@deepseek-ai/dsh-session-persistence-jsonl/lib/index.js`);
const { Context } = await import(`file://${UNPACKED}/@deepseek-ai/cordis/lib/index.js`);

const ctx = new Context();
// The persistence coordinator's write-path installer iterates ctx.sessions.list().
// Stub the service: we only exercise reads, and a bare context has no sessions.
Object.defineProperty(ctx, "sessions", { value: { list: () => [] }, configurable: true, writable: true });
const persistence = new JsonlSessionPersistence(ctx, {
  root,
  packChunks: true,
  compression: "zstd",
  preparedSessionCacheSize: 64,
  writeBatchMaxDelayMs: 1000,
});
console.log("persistence instantiated:", persistence.name, "compression:", persistence.compression);

// 1) The exact history-load path: readPrefix
const prefix = await persistence.readPrefix(logPath, sessionId);
const last = prefix.events[prefix.events.length - 1];
console.log("\nreadPrefix OK — events:", prefix.events.length, "| last seq:", last.seq, "| last type:", last.type, "| tornMarker:", prefix.tornMarker ?? "none");

// 2) raw read (byte-verbatim content)
const raw = await persistence.readRaw(sessionId);
console.log("readRaw OK — id:", raw.meta.id, "| content bytes:", raw.content.length);
console.log("  last raw line tail:", raw.content.trimEnd().slice(-160));

// 3) session listing (header-only read used by the session picker)
const sessions = await persistence.list();
const found = sessions.find((s) => s.id === sessionId);
console.log("list OK — sessions:", sessions.length, "| target found:", found !== undefined, "| id match:", found?.id === sessionId, "| cwd:", found?.cwd, "| preset:", found?.agentPreset);

// 4) contiguity + first/last event identity
const first = prefix.events[0];
let contiguous = true;
for (let i = 0; i < prefix.events.length; i++) {
  if (prefix.events[i].seq !== i) { contiguous = false; console.log("gap at", i, "got", prefix.events[i].seq); break; }
}
console.log("contiguous 0..N:", contiguous);
console.log("first event:", JSON.stringify({ type: first.type, seq: first.seq, time: first.time }));
process.exit(0);