import { useState, useEffect, useCallback, useRef } from "react";

// ─── API CONFIG ──────────────────────────────────────────────────────────────
// Change this single variable to point at a different backend host.
const API_BASE = "http://localhost:4235";

// ─── API HELPERS ─────────────────────────────────────────────────────────────
const api = {
  submit: (body) =>
    fetch(`${API_BASE}/api`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      redirect: "follow",
    }).then((r) => r.json()),

  status: (hash) =>
    fetch(`${API_BASE}/status?hash=${encodeURIComponent(hash)}`, {
      redirect: "follow",
    }).then((r) => r.json()),

  file: (path) =>
    fetch(`${API_BASE}/file?path=${encodeURIComponent(path)}`, {
      redirect: "follow",
    }).then((r) => r.json()),

  files: () =>
    fetch(`${API_BASE}/files`, { redirect: "follow" }).then((r) => r.json()),
};

// ─── POLLING UTILITY ─────────────────────────────────────────────────────────
async function pollUntilSettled(hash, maxMs = 8000, intervalMs = 600) {
  const start = Date.now();
  while (Date.now() - start < maxMs) {
    const res = await api.status(hash);
    if (res.status === "success") return { ok: true };
    if (res.status === "failed") return { ok: false };
    await new Promise((r) => setTimeout(r, intervalMs));
  }
  return { ok: false, timeout: true };
}

// ─── TREE BUILDER ────────────────────────────────────────────────────────────
// Returns a nested node tree: { type: "dir"|"file", name, path?, children? }
function buildTree(files) {
  const imageExts = [".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp"];
  const root = {}; // nested plain object used as trie

  for (const f of files) {
    const lower = f.toLowerCase();
    if (lower.endsWith(".meta.json")) continue;
    if (!imageExts.some((e) => lower.endsWith(e))) continue;

    const parts = f.split("/");
    let node = root;
    for (let i = 0; i < parts.length - 1; i++) {
      const seg = parts[i];
      if (!node[seg]) node[seg] = {};
      node = node[seg];
    }
    // Mark the leaf with the full path using a sentinel key
    node[`\0${parts[parts.length - 1]}`] = f;
  }

  function toNodes(obj, prefix = "") {
    const dirs = [];
    const leaves = [];
    for (const key of Object.keys(obj).sort()) {
      if (key.startsWith("\0")) {
        leaves.push({ type: "file", name: key.slice(1), path: obj[key] });
      } else {
        dirs.push({
          type: "dir",
          name: key,
          children: toNodes(obj[key], prefix ? `${prefix}/${key}` : key),
        });
      }
    }
    return [...dirs, ...leaves];
  }

  return toNodes(root);
}

// ─── RAFT DIAGRAM MODAL ───────────────────────────────────────────────────────
const RAFT_SVG = `
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 640 340" font-family="'Georgia',serif">
  <defs>
    <marker id="arr" markerWidth="8" markerHeight="6" refX="7" refY="3" orient="auto">
      <polygon points="0 0, 8 3, 0 6" fill="#94a3b8"/>
    </marker>
    <marker id="arr-w" markerWidth="8" markerHeight="6" refX="7" refY="3" orient="auto">
      <polygon points="0 0, 8 3, 0 6" fill="#e2e8f0"/>
    </marker>
    <filter id="glow">
      <feGaussianBlur stdDeviation="3" result="blur"/>
      <feComposite in="SourceGraphic" in2="blur" operator="over"/>
    </filter>
  </defs>
  <!-- Background -->
  <rect width="640" height="340" fill="#0f172a" rx="12"/>

  <!-- FOLLOWER -->
  <ellipse cx="100" cy="170" rx="72" ry="40" fill="#1e293b" stroke="#475569" stroke-width="1.5"/>
  <text x="100" y="165" text-anchor="middle" fill="#e2e8f0" font-size="13" font-weight="bold">Follower</text>
  <text x="100" y="182" text-anchor="middle" fill="#94a3b8" font-size="9">receives heartbeats</text>

  <!-- CANDIDATE -->
  <ellipse cx="320" cy="80" rx="80" ry="40" fill="#1e293b" stroke="#475569" stroke-width="1.5"/>
  <text x="320" y="75" text-anchor="middle" fill="#e2e8f0" font-size="13" font-weight="bold">Candidate</text>
  <text x="320" y="92" text-anchor="middle" fill="#94a3b8" font-size="9">requests votes</text>

  <!-- LEADER -->
  <ellipse cx="540" cy="170" rx="72" ry="40" fill="#1e293b" stroke="#3b82f6" stroke-width="2" filter="url(#glow)"/>
  <text x="540" y="165" text-anchor="middle" fill="#93c5fd" font-size="13" font-weight="bold">Leader</text>
  <text x="540" y="182" text-anchor="middle" fill="#60a5fa" font-size="9">sends heartbeats</text>

  <!-- Follower → Candidate: timeout -->
  <path d="M 158 145 Q 230 60 242 80" stroke="#94a3b8" stroke-width="1.4" fill="none" marker-end="url(#arr)"/>
  <text x="186" y="90" fill="#64748b" font-size="9" text-anchor="middle">election timeout</text>
  <text x="186" y="101" fill="#64748b" font-size="9" text-anchor="middle">starts election</text>

  <!-- Candidate → Candidate: split vote -->
  <path d="M 370 62 Q 430 20 390 62" stroke="#94a3b8" stroke-width="1.4" fill="none" marker-end="url(#arr)"/>
  <text x="420" y="30" fill="#64748b" font-size="9" text-anchor="middle">split vote /</text>
  <text x="420" y="41" fill="#64748b" font-size="9" text-anchor="middle">new election</text>

  <!-- Candidate → Leader: wins election -->
  <path d="M 395 95 Q 470 100 472 138" stroke="#3b82f6" stroke-width="1.6" fill="none" marker-end="url(#arr-w)"/>
  <text x="462" y="110" fill="#60a5fa" font-size="9" text-anchor="middle">receives majority</text>
  <text x="462" y="121" fill="#60a5fa" font-size="9" text-anchor="middle">of votes</text>

  <!-- Leader → Follower: discovers higher term -->
  <path d="M 540 210 Q 540 280 150 210" stroke="#94a3b8" stroke-width="1.4" fill="none" marker-end="url(#arr)"/>
  <text x="345" y="278" fill="#64748b" font-size="9" text-anchor="middle">discovers server with higher term</text>

  <!-- Candidate → Follower: discovers current leader -->
  <path d="M 248 105 Q 190 150 170 155" stroke="#94a3b8" stroke-width="1.4" fill="none" marker-end="url(#arr)"/>
  <text x="185" y="140" fill="#64748b" font-size="9" text-anchor="middle">discovers</text>
  <text x="185" y="151" fill="#64748b" font-size="9" text-anchor="middle">current leader</text>

  <!-- Caption -->
  <text x="320" y="322" text-anchor="middle" fill="#334155" font-size="9.5">
    Raft Consensus — Server State Transitions (Ongaro &amp; Ousterhout, 2014)
  </text>
</svg>
`;

// ─── STATUS BADGE ─────────────────────────────────────────────────────────────
function OpStatus({ op }) {
  if (!op) return null;
  const map = {
    pending:  { bg: "bg-amber-950", text: "text-amber-300",  ring: "ring-amber-800",  label: "pending"  },
    success:  { bg: "bg-emerald-950",text: "text-emerald-300",ring: "ring-emerald-800",label: "committed" },
    failed:   { bg: "bg-red-950",   text: "text-red-300",    ring: "ring-red-800",    label: "failed"   },
    rolling:  { bg: "bg-orange-950",text: "text-orange-300", ring: "ring-orange-800", label: "rolling back…" },
  };
  const s = map[op.state] ?? map.pending;
  return (
    <div className={`mt-3 rounded-md px-3 py-2 text-xs ring-1 ${s.bg} ${s.text} ${s.ring} font-mono`}>
      {op.state === "pending" && (
        <span className="mr-2 inline-block animate-spin">⟳</span>
      )}
      <span className="font-semibold uppercase tracking-widest">{s.label}</span>
      {op.hash && <span className="ml-2 opacity-50">#{op.hash.slice(0, 12)}</span>}
      {op.msg && <div className="mt-1 opacity-70">{op.msg}</div>}
    </div>
  );
}

// ─── TREE SIDEBAR ─────────────────────────────────────────────────────────────
function TreeNode({ node, depth, selected, onSelect }) {
  const [open, setOpen] = useState(true);
  const indent = depth * 12;

  if (node.type === "file") {
    return (
      <li>
        <button
          onClick={() => onSelect(node.path)}
          className={`w-full text-left py-1.5 text-xs font-mono truncate transition-colors flex items-center gap-1.5
            ${selected === node.path
              ? "bg-slate-800 text-slate-100"
              : "text-slate-400 hover:bg-slate-900 hover:text-slate-200"
            }`}
          style={{ paddingLeft: 16 + indent }}
          title={node.path}
        >
          <span className="text-slate-600 shrink-0">🖼</span>
          <span className="truncate">{node.name}</span>
        </button>
      </li>
    );
  }

  // Directory node
  return (
    <li>
      <button
        onClick={() => setOpen((o) => !o)}
        className="w-full text-left py-1.5 text-xs font-mono text-slate-500 hover:text-slate-300 transition-colors flex items-center gap-1.5"
        style={{ paddingLeft: 16 + indent }}
      >
        <span className="text-slate-600 shrink-0 text-[10px]">{open ? "▾" : "▸"}</span>
        <span className="text-slate-400">{node.name}/</span>
      </button>
      {open && (
        <ul>
          {node.children.map((child) => (
            <TreeNode
              key={child.type === "file" ? child.path : child.name}
              node={child}
              depth={depth + 1}
              selected={selected}
              onSelect={onSelect}
            />
          ))}
        </ul>
      )}
    </li>
  );
}

function Sidebar({ files, selected, onSelect, onRefresh }) {
  const tree = buildTree(files);
  return (
    <aside className="w-56 shrink-0 border-r border-slate-800 flex flex-col bg-slate-950">
      <div className="flex items-center justify-between px-4 py-3 border-b border-slate-800">
        <span className="text-xs font-semibold text-slate-400 uppercase tracking-widest">Files</span>
        <button
          onClick={onRefresh}
          className="text-slate-500 hover:text-slate-200 transition-colors text-sm"
          title="Refresh"
        >⟳</button>
      </div>
      <ul className="flex-1 overflow-y-auto py-2">
        {tree.length === 0 && (
          <li className="px-4 py-2 text-xs text-slate-600 italic">No images yet.</li>
        )}
        {tree.map((node) => (
          <TreeNode
            key={node.type === "file" ? node.path : node.name}
            node={node}
            depth={0}
            selected={selected}
            onSelect={onSelect}
          />
        ))}
      </ul>
    </aside>
  );
}

// ─── METADATA FIELDS ─────────────────────────────────────────────────────────
function MetaEditor({ meta, onChange }) {
  const [pairs, setPairs] = useState(() =>
    Object.entries(meta).map(([k, v]) => ({ k, v }))
  );

  useEffect(() => {
    const obj = {};
    pairs.forEach(({ k, v }) => { if (k.trim()) obj[k.trim()] = v; });
    onChange(obj);
  }, [pairs]);

  const add = () => setPairs((p) => [...p, { k: "", v: "" }]);
  const remove = (i) => setPairs((p) => p.filter((_, j) => j !== i));
  const update = (i, field, val) =>
    setPairs((p) => p.map((item, j) => (j === i ? { ...item, [field]: val } : item)));

  return (
    <div className="space-y-2">
      {pairs.map(({ k, v }, i) => (
        <div key={i} className="flex gap-2">
          <input
            className="flex-1 bg-slate-900 border border-slate-700 rounded px-2 py-1 text-xs font-mono text-slate-200 placeholder-slate-600 focus:outline-none focus:border-slate-500"
            placeholder="key"
            value={k}
            onChange={(e) => update(i, "k", e.target.value)}
          />
          <input
            className="flex-[2] bg-slate-900 border border-slate-700 rounded px-2 py-1 text-xs font-mono text-slate-200 placeholder-slate-600 focus:outline-none focus:border-slate-500"
            placeholder="value"
            value={v}
            onChange={(e) => update(i, "v", e.target.value)}
          />
          <button
            onClick={() => remove(i)}
            className="text-slate-600 hover:text-red-400 transition-colors text-xs px-1"
          >✕</button>
        </div>
      ))}
      <button
        onClick={add}
        className="text-xs text-slate-500 hover:text-slate-300 transition-colors border border-dashed border-slate-700 hover:border-slate-500 rounded px-3 py-1 w-full"
      >+ add field</button>
    </div>
  );
}

// ─── UPLOAD PANEL ─────────────────────────────────────────────────────────────
function UploadPanel({ onSuccess }) {
  const [file, setFile] = useState(null);
  const [customPath, setCustomPath] = useState("");
  const [meta, setMeta] = useState({});
  const [op, setOp] = useState(null);
  const [busy, setBusy] = useState(false);
  const fileRef = useRef();

  const handleFile = (e) => {
    const f = e.target.files?.[0];
    if (f) {
      setFile(f);
      // Pre-populate path with the filename; user can edit freely
      setCustomPath((prev) => prev || f.name);
    }
  };

  const readAsBase64 = (f) =>
    new Promise((res, rej) => {
      const r = new FileReader();
      r.onload = () => res(r.result);
      r.onerror = rej;
      r.readAsDataURL(f);
    });

  const submit = async () => {
    if (!file) return;
    const trimmedPath = customPath.trim();
    if (!trimmedPath) return;
    setBusy(true);
    setOp({ state: "pending", msg: "Submitting image…" });

    // Use the user-supplied path; derive meta path from it
    const imgPath  = trimmedPath;
    const stem     = trimmedPath.replace(/\.[^.]+$/, "");
    const metaPath = `${stem}.meta.json`;

    const b64 = await readAsBase64(file);

    let imgHash, metaHash;

    // 1. Submit image
    try {
      const imgRes = await api.submit({
        type: "data",
        message: { path: imgPath, blob: b64, operation: "write" },
      });
      imgHash = imgRes.hash;
    } catch {
      setOp({ state: "failed", msg: "Network error submitting image." });
      setBusy(false);
      return;
    }

    setOp({ state: "pending", hash: imgHash, msg: "Awaiting image commit…" });
    const imgOk = await pollUntilSettled(imgHash);

    if (!imgOk.ok) {
      setOp({ state: "failed", hash: imgHash, msg: "Image failed to commit." });
      setBusy(false);
      return;
    }

    // 2. Submit metadata
    const metaPayload = { ...meta, _imagePath: imgPath, _uploadedAt: new Date().toISOString() };
    try {
      const metaRes = await api.submit({
        type: "data",
        message: { path: metaPath, blob: JSON.stringify(metaPayload, null, 2), operation: "write" },
      });
      metaHash = metaRes.hash;
    } catch {
      // Rollback image
      setOp({ state: "rolling", msg: "Meta submission failed — rolling back image…" });
      await api.submit({ type: "data", message: { path: imgPath, blob: "", operation: "delete" } });
      setOp({ state: "failed", msg: "Rolled back. Metadata could not be submitted." });
      setBusy(false);
      return;
    }

    setOp({ state: "pending", hash: metaHash, msg: "Awaiting metadata commit…" });
    const metaOk = await pollUntilSettled(metaHash);

    if (!metaOk.ok) {
      // Rollback image
      setOp({ state: "rolling", msg: "Metadata failed — rolling back image…" });
      await api.submit({ type: "data", message: { path: imgPath, blob: "", operation: "delete" } });
      setOp({ state: "failed", msg: "Rolled back. Both files removed." });
      setBusy(false);
      return;
    }

    setOp({ state: "success", hash: metaHash });
    setBusy(false);
    setFile(null);
    setCustomPath("");
    setMeta({});
    if (fileRef.current) fileRef.current.value = "";
    onSuccess?.();
  };

  return (
    <div className="bg-slate-900 rounded-lg border border-slate-800 p-5 space-y-4">
      <h2 className="text-xs font-semibold text-slate-400 uppercase tracking-widest">Upload Image</h2>

      {/* File drop zone */}
      <label
        className={`flex flex-col items-center justify-center h-28 rounded-lg border-2 border-dashed cursor-pointer transition-colors
          ${file ? "border-slate-600 bg-slate-800" : "border-slate-700 hover:border-slate-500 bg-slate-900"}`}
      >
        {file ? (
          <div className="text-center">
            <div className="text-slate-200 text-sm font-mono">{file.name}</div>
            <div className="text-slate-500 text-xs mt-1">{(file.size / 1024).toFixed(1)} KB</div>
          </div>
        ) : (
          <div className="text-center text-slate-500 text-xs">
            <div className="text-2xl mb-1">🖼</div>
            click or drag an image
          </div>
        )}
        <input ref={fileRef} type="file" accept="image/*" className="hidden" onChange={handleFile} />
      </label>

      {/* Path */}
      <div>
        <label className="text-xs text-slate-500 mb-1.5 uppercase tracking-widest block">
          Save path
        </label>
        <div className="flex items-center bg-slate-900 border border-slate-700 rounded focus-within:border-slate-500 transition-colors">
          <span className="pl-2.5 text-slate-600 text-xs font-mono select-none shrink-0">data/</span>
          <input
            className="flex-1 bg-transparent py-1.5 pr-2.5 text-xs font-mono text-slate-200 placeholder-slate-600 focus:outline-none"
            placeholder="folder/image.png"
            value={customPath}
            onChange={(e) => setCustomPath(e.target.value)}
            spellCheck={false}
          />
        </div>
        <p className="mt-1 text-xs text-slate-600">Use <span className="font-mono text-slate-500">/</span> to create folders, e.g. <span className="font-mono text-slate-500">animals/cat.png</span></p>
      </div>

      {/* Metadata */}
      <div>
        <div className="text-xs text-slate-500 mb-2 uppercase tracking-widest">Metadata</div>
        <MetaEditor meta={meta} onChange={setMeta} />
      </div>

      <button
        onClick={submit}
        disabled={!file || !customPath.trim() || busy}
        className="w-full py-2 rounded bg-slate-700 hover:bg-slate-600 disabled:opacity-30 disabled:cursor-not-allowed text-slate-200 text-xs font-semibold uppercase tracking-widest transition-colors"
      >
        {busy ? "Uploading…" : "Upload"}
      </button>

      <OpStatus op={op} />
    </div>
  );
}

// ─── FILE DETAIL PANEL ────────────────────────────────────────────────────────
function FileDetail({ path, onDeleted, onRefresh }) {
  const [imgSrc, setImgSrc] = useState(null);
  const [imgRaw, setImgRaw] = useState(null); // raw b64 for re-upload
  const [metaObj, setMetaObj] = useState({});
  const [editMeta, setEditMeta] = useState({});
  const [loading, setLoading] = useState(true);
  const [op, setOp] = useState(null);
  const [editing, setEditing] = useState(false);
  const replaceRef = useRef();

  const stem = path.replace(/\.[^.]+$/, "");
  const ext  = path.slice(stem.length);
  const metaPath = `${stem}.meta.json`;

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const raw = await api.file(path);
      setImgSrc(raw);
      setImgRaw(raw);
    } catch { setImgSrc(null); }
    try {
      const rawMeta = await api.file(metaPath);
      const parsed = JSON.parse(rawMeta);
      setMetaObj(parsed);
      setEditMeta(parsed);
    } catch { setMetaObj({}); setEditMeta({}); }
    setLoading(false);
  }, [path]);

  useEffect(() => { load(); setOp(null); setEditing(false); }, [path]);

  const downloadImage = () => {
    if (!imgSrc) return;
    const a = document.createElement("a");
    a.href = imgSrc.startsWith("data:") ? imgSrc : `data:image/png;base64,${imgSrc}`;
    a.download = path.split("/").pop();
    a.click();
  };

  const saveMetaUpdate = async () => {
    setOp({ state: "pending", msg: "Saving metadata…" });
    const blob = JSON.stringify(editMeta, null, 2);
    try {
      const res = await api.submit({
        type: "data",
        message: { path: metaPath, blob, operation: "update" },
      });
      const result = await pollUntilSettled(res.hash);
      if (result.ok) {
        setOp({ state: "success", hash: res.hash });
        setMetaObj(editMeta);
        setEditing(false);
        onRefresh?.();
      } else {
        setOp({ state: "failed", msg: "Metadata update failed to commit." });
      }
    } catch {
      setOp({ state: "failed", msg: "Network error." });
    }
  };

  const deleteAll = async () => {
    if (!window.confirm(`Delete ${path} and its metadata?`)) return;
    setOp({ state: "pending", msg: "Deleting image…" });

    // Delete image
    let imgHash;
    try {
      const r = await api.submit({ type: "data", message: { path, blob: "", operation: "delete" } });
      imgHash = r.hash;
    } catch {
      setOp({ state: "failed", msg: "Failed to submit image deletion." });
      return;
    }
    const imgOk = await pollUntilSettled(imgHash);
    if (!imgOk.ok) {
      setOp({ state: "failed", msg: "Image deletion failed to commit." });
      return;
    }

    // Delete meta
    setOp({ state: "pending", msg: "Deleting metadata…" });
    try {
      const r2 = await api.submit({ type: "data", message: { path: metaPath, blob: "", operation: "delete" } });
      const metaOk = await pollUntilSettled(r2.hash);
      if (!metaOk.ok) {
        setOp({ state: "failed", msg: "Metadata deletion failed (image removed)." });
        return;
      }
    } catch { /* metadata may not exist, that's okay */ }

    setOp({ state: "success" });
    onDeleted?.();
    onRefresh?.();
  };

  const replaceImage = async (e) => {
    const f = e.target.files?.[0];
    if (!f) return;
    setOp({ state: "pending", msg: "Uploading replacement image…" });

    const b64 = await new Promise((res, rej) => {
      const r = new FileReader();
      r.onload = () => res(r.result);
      r.onerror = rej;
      r.readAsDataURL(f);
    });

    try {
      const res = await api.submit({
        type: "data",
        message: { path, blob: b64, operation: "update" },
      });
      const result = await pollUntilSettled(res.hash);
      if (result.ok) {
        setOp({ state: "success", hash: res.hash });
        setImgSrc(b64);
        onRefresh?.();
      } else {
        setOp({ state: "failed", msg: "Image replacement failed to commit." });
      }
    } catch {
      setOp({ state: "failed", msg: "Network error." });
    }
    if (replaceRef.current) replaceRef.current.value = "";
  };

  if (loading) return <div className="text-slate-600 text-xs p-6">Loading…</div>;

  const displayMeta = Object.fromEntries(
    Object.entries(metaObj).filter(([k]) => !k.startsWith("_"))
  );

  return (
    <div className="space-y-5">
      {/* Image */}
      <div className="rounded-lg overflow-hidden bg-slate-900 border border-slate-800 flex items-center justify-center" style={{ minHeight: 200 }}>
        {imgSrc ? (
          <img
            src={imgSrc.startsWith("data:") ? imgSrc : `data:image/png;base64,${imgSrc}`}
            alt={path}
            className="max-w-full max-h-72 object-contain"
          />
        ) : (
          <span className="text-slate-600 text-sm">Image unavailable</span>
        )}
      </div>

      {/* Actions */}
      <div className="flex flex-wrap gap-2">
        <button
          onClick={downloadImage}
          className="px-3 py-1.5 text-xs rounded bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 transition-colors"
        >⬇ Download</button>

        <label className="px-3 py-1.5 text-xs rounded bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 cursor-pointer transition-colors">
          ↺ Replace
          <input ref={replaceRef} type="file" accept="image/*" className="hidden" onChange={replaceImage} />
        </label>

        <button
          onClick={deleteAll}
          className="px-3 py-1.5 text-xs rounded bg-red-950 hover:bg-red-900 text-red-300 border border-red-900 transition-colors"
        >🗑 Delete</button>
      </div>

      {/* Metadata */}
      <div className="bg-slate-900 rounded-lg border border-slate-800 p-4 space-y-3">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold text-slate-400 uppercase tracking-widest">Metadata</span>
          {!editing
            ? <button onClick={() => setEditing(true)} className="text-xs text-slate-500 hover:text-slate-300 transition-colors">Edit</button>
            : <div className="flex gap-2">
                <button onClick={saveMetaUpdate} className="text-xs text-emerald-400 hover:text-emerald-300 transition-colors">Save</button>
                <button onClick={() => { setEditing(false); setEditMeta(metaObj); }} className="text-xs text-slate-500 hover:text-slate-300 transition-colors">Cancel</button>
              </div>
          }
        </div>

        {editing ? (
          <MetaEditor meta={editMeta} onChange={setEditMeta} />
        ) : (
          <dl className="space-y-1">
            {Object.keys(displayMeta).length === 0 && (
              <span className="text-xs text-slate-600 italic">No metadata.</span>
            )}
            {Object.entries(displayMeta).map(([k, v]) => (
              <div key={k} className="flex gap-2 text-xs font-mono">
                <dt className="text-slate-500 shrink-0 min-w-16">{k}</dt>
                <dd className="text-slate-300 break-all">{String(v)}</dd>
              </div>
            ))}
            {metaObj._uploadedAt && (
              <div className="flex gap-2 text-xs font-mono mt-2 pt-2 border-t border-slate-800">
                <dt className="text-slate-600 shrink-0 min-w-16">uploaded</dt>
                <dd className="text-slate-600">{new Date(metaObj._uploadedAt).toLocaleString()}</dd>
              </div>
            )}
          </dl>
        )}
      </div>

      <OpStatus op={op} />
    </div>
  );
}

// ─── RAFT MODAL ───────────────────────────────────────────────────────────────
function RaftModal({ open, onClose }) {
  if (!open) return null;
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="relative bg-slate-900 rounded-xl border border-slate-700 shadow-2xl max-w-2xl w-full mx-4 overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between px-5 py-3 border-b border-slate-800">
          <div>
            <h2 className="text-sm font-semibold text-slate-200">Raft State Transitions</h2>
            <p className="text-xs text-slate-500 mt-0.5">Ongaro & Ousterhout, 2014 — "In Search of an Understandable Consensus Algorithm"</p>
          </div>
          <button onClick={onClose} className="text-slate-500 hover:text-slate-200 text-lg transition-colors">✕</button>
        </div>
        <div className="p-4">
          <div
            dangerouslySetInnerHTML={{ __html: RAFT_SVG }}
            className="w-full"
          />
        </div>
        <div className="px-5 py-3 border-t border-slate-800 text-xs text-slate-600 space-y-1">
          <p><span className="text-slate-400 font-semibold">Follower</span> — starts here; converts to candidate on election timeout.</p>
          <p><span className="text-slate-400 font-semibold">Candidate</span> — requests votes; becomes leader on majority, or reverts on higher-term discovery.</p>
          <p><span className="text-slate-400 font-semibold">Leader</span> — sends heartbeats; steps down if it discovers a higher term.</p>
          <p className="pt-2 border-t border-slate-800">
            <a
              href="https://raft.github.io/raft.pdf"
              target="_blank"
              rel="noopener noreferrer"
              className="text-slate-500 hover:text-slate-300 underline underline-offset-2 transition-colors"
            >
              Ongaro &amp; Ousterhout, 2014 — "In Search of an Understandable Consensus Algorithm" ↗
            </a>
          </p>
        </div>
      </div>
    </div>
  );
}

// ─── APP ROOT ─────────────────────────────────────────────────────────────────
export default function App() {
  const [files, setFiles] = useState([]);
  const [selected, setSelected] = useState(null);
  const [raftOpen, setRaftOpen] = useState(false);

  const refreshFiles = useCallback(async () => {
    try {
      const f = await api.files();
      setFiles(Array.isArray(f) ? f : []);
    } catch { setFiles([]); }
  }, []);

  useEffect(() => { refreshFiles(); }, []);

  const handleDeleted = () => setSelected(null);

  return (
    <div className="min-h-screen bg-slate-950 text-slate-200 flex flex-col" style={{ fontFamily: "'IBM Plex Mono', 'Courier New', monospace" }}>

      {/* Header */}
      <header className="border-b border-slate-800 px-6 py-3 flex items-center gap-4">
        <div className="flex items-center gap-3">
          <h1 className="text-lg font-bold tracking-tight text-slate-100">gorp</h1>
          <a
            href="https://github.com/dannowilby/gorp/actions/workflows/go.yml"
            target="_blank"
            rel="noopener noreferrer"
          >
            <img
              src="https://github.com/dannowilby/gorp/actions/workflows/go.yml/badge.svg"
              alt="Go CI"
              className="h-5"
            />
          </a>
        </div>
        <div className="flex-1" />
        <button
          onClick={() => setRaftOpen(true)}
          className="text-xs text-slate-500 hover:text-slate-300 border border-slate-800 hover:border-slate-600 rounded px-3 py-1.5 transition-colors"
        >
          Raft diagram
        </button>
      </header>

      {/* Body */}
      <div className="flex flex-1 overflow-hidden">
        <Sidebar
          files={files}
          selected={selected}
          onSelect={setSelected}
          onRefresh={refreshFiles}
        />

        <main className="flex-1 overflow-y-auto p-6 max-w-2xl">
          {/* Detail takes over when a file is selected; upload shown otherwise */}
          {selected ? (
            <div className="bg-slate-900 rounded-lg border border-slate-800 p-5">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-xs font-semibold text-slate-400 uppercase tracking-widest truncate">
                  {selected.split("/").pop()}
                </h2>
                <button
                  onClick={() => setSelected(null)}
                  className="text-slate-600 hover:text-slate-300 text-xs ml-2 transition-colors"
                >✕ close</button>
              </div>
              <FileDetail
                path={selected}
                onDeleted={handleDeleted}
                onRefresh={refreshFiles}
              />
            </div>
          ) : (
            <UploadPanel onSuccess={refreshFiles} />
          )}
        </main>
      </div>

      <RaftModal open={raftOpen} onClose={() => setRaftOpen(false)} />
    </div>
  );
}