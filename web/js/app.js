// ghdiff - Frontend Application
// Vanilla JS, no build step, no framework.

(function () {
  "use strict";

  // --- State ---
  let currentFiles = [];
  let viewMode = "split"; // "split" or "unified"
  let activeFile = null;
  var fileContextLevels = {}; // filePath -> current context level (0 = default git context)

  // --- DOM References ---
  const basePicker = document.getElementById("base-picker");
  const targetPicker = document.getElementById("target-picker");
  const btnSplit = document.getElementById("btn-split");
  const btnUnified = document.getElementById("btn-unified");
  const fileTreeContent = document.getElementById("file-tree-content");
  const diffContent = document.getElementById("diff-content");

  // --- Auth Token ---
  const authHeaders = { "X-Auth-Token": window.__TOKEN__ };

  // --- API Calls ---

  async function fetchDiff(base, target) {
    const params = new URLSearchParams();
    if (base) params.set("base", base);
    if (target) params.set("target", target);
    const qs = params.toString();
    const url = qs ? `/api/diff?${qs}` : "/api/diff";
    const resp = await fetch(url, { headers: authHeaders });
    if (!resp.ok) {
      throw new Error(`Failed to fetch diff: ${resp.status} ${resp.statusText}`);
    }
    return resp.json();
  }

  async function fetchExpand(file, context, base, target) {
    const params = new URLSearchParams();
    params.set("file", file);
    params.set("context", context);
    if (base) params.set("base", base);
    if (target) params.set("target", target);
    const resp = await fetch(`/api/expand?${params.toString()}`, { headers: authHeaders });
    if (!resp.ok) {
      throw new Error(`Failed to expand: ${resp.status} ${resp.statusText}`);
    }
    return resp.json();
  }

  async function fetchCommits() {
    const resp = await fetch("/api/commits", { headers: authHeaders });
    if (!resp.ok) {
      throw new Error(
        `Failed to fetch commits: ${resp.status} ${resp.statusText}`
      );
    }
    return resp.json();
  }

  // --- File Tree ---

  function buildFileTree(files) {
    const root = { name: "", children: new Map(), files: [] };

    for (const file of files) {
      const path =
        file.status === "deleted" ? file.oldName : file.newName || file.oldName;
      const parts = path.split("/");
      let current = root;

      for (let i = 0; i < parts.length - 1; i++) {
        const dir = parts[i];
        if (!current.children.has(dir)) {
          current.children.set(dir, { name: dir, children: new Map(), files: [] });
        }
        current = current.children.get(dir);
      }

      current.files.push({
        name: parts[parts.length - 1],
        path: path,
        status: file.status,
      });
    }

    return root;
  }

  function renderFileTree(files) {
    fileTreeContent.innerHTML = "";
    if (!files || files.length === 0) {
      fileTreeContent.innerHTML =
        '<div class="loading">No files changed</div>';
      return;
    }
    const tree = buildFileTree(files);
    const fragment = document.createDocumentFragment();
    renderTreeNode(tree, fragment, 0, true);
    fileTreeContent.appendChild(fragment);
  }

  function renderTreeNode(node, parent, depth, isRoot) {
    // Render folders sorted alphabetically
    const sortedChildren = Array.from(node.children.entries()).sort((a, b) =>
      a[0].localeCompare(b[0])
    );

    for (const [, child] of sortedChildren) {
      const folder = document.createElement("div");
      folder.className = "tree-folder";

      const label = document.createElement("div");
      label.className = "tree-folder-label";
      label.style.setProperty("--indent-pad", `${12 + depth * 16}px`);

      label.innerHTML = `
        <span class="arrow">&#9660;</span>
        <span class="folder-icon">&#128193;</span>
        <span class="folder-name">${escapeHtml(child.name)}</span>
      `;

      label.addEventListener("click", () => {
        folder.classList.toggle("collapsed");
      });

      folder.appendChild(label);

      const children = document.createElement("div");
      children.className = "tree-folder-children";
      renderTreeNode(child, children, depth + 1, false);
      folder.appendChild(children);

      parent.appendChild(folder);
    }

    // Render files sorted alphabetically
    const sortedFiles = [...node.files].sort((a, b) =>
      a.name.localeCompare(b.name)
    );

    for (const file of sortedFiles) {
      const el = document.createElement("div");
      el.className = "tree-file";
      el.dataset.path = file.path;
      el.style.setProperty("--indent-pad", `${(isRoot ? 12 : 28) + depth * 16}px`);

      const statusChar = {
        added: "+",
        modified: "M",
        deleted: "\u2212",
        renamed: "R",
      }[file.status] || "?";

      el.innerHTML = `
        <span class="status-indicator ${file.status}">${statusChar}</span>
        <span class="file-name" title="${escapeHtml(file.path)}">${escapeHtml(file.name)}</span>
      `;

      el.addEventListener("click", () => {
        scrollToFile(file.path);
        setActiveTreeFile(file.path);
      });

      parent.appendChild(el);
    }
  }

  function setActiveTreeFile(path) {
    activeFile = path;
    const allFiles = fileTreeContent.querySelectorAll(".tree-file");
    for (const f of allFiles) {
      f.classList.toggle("active", f.dataset.path === path);
    }
  }

  // --- Expand Context ---

  // Returns the next context level for a file. Each call doubles the level:
  // 0 (default) -> 10 -> 20 -> 40 -> 80 -> 160 -> ... up to all (999999).
  // "all" always returns the full file.
  function nextContextLevel(filePath, action) {
    if (action === "expand-all") return "all";
    var level = fileContextLevels[filePath] || 0;
    level++;
    fileContextLevels[filePath] = level;
    // Double each time: 10, 20, 40, 80, 160, 320, 640, 1280, 2560, 5120, 10240, ...
    var ctx = 10 * Math.pow(2, level - 1);
    if (ctx >= 999999) return "all";
    return String(Math.round(ctx));
  }

  async function expandFileContext(filePath, action, evt) {
    var context;
    if (action === "reset") {
      // Reset to default context
      fileContextLevels[filePath] = 0;
      context = "0";
    } else {
      context = nextContextLevel(filePath, action);
    }
    const base = basePicker.value || undefined;
    const target = targetPicker.value || undefined;
    try {
      const data = await fetchExpand(filePath, context, base, target);
      if (!data.hunks || data.hunks.length === 0) return;

      // Find and update the file in currentFiles
      const fileObj = currentFiles.find(function (f) {
        const p = f.status === "deleted" ? f.oldName : f.newName || f.oldName;
        return p === filePath;
      });
      if (fileObj) {
        fileObj.hunks = data.hunks;
      }

      // Re-render just this file section in-place
      const sectionEl = document.getElementById("file-" + cssId(filePath));
      if (sectionEl && fileObj) {
        const oldBody = sectionEl.querySelector(".file-body");
        const newBody = document.createElement("div");
        newBody.className = "file-body";
        buildFileBody(fileObj, newBody);
        if (oldBody) {
          oldBody.replaceWith(newBody);
        } else {
          sectionEl.appendChild(newBody);
        }
        adjustSplitTableWidths();

        // Update stats in header
        const statsEl = sectionEl.querySelector(".change-stats");
        if (statsEl) {
          statsEl.innerHTML = buildChangeStats(fileObj);
        }
      }
    } catch (err) {
      console.error("Expand failed:", err);
    }
  }

  function buildChangeStats(file) {
    var additions = 0;
    var deletions = 0;
    if (file.hunks) {
      for (var h = 0; h < file.hunks.length; h++) {
        var lines = file.hunks[h].lines;
        for (var l = 0; l < lines.length; l++) {
          if (lines[l].type === "add") additions++;
          if (lines[l].type === "delete") deletions++;
        }
      }
    }
    var parts = [];
    if (additions > 0) parts.push('<span class="additions">+' + additions + "</span>");
    if (deletions > 0) parts.push('<span class="deletions">&minus;' + deletions + "</span>");
    if (parts.length > 0) {
      return '<span class="change-stats">' + parts.join(" ") + "</span>";
    }
    return "";
  }

  // --- Diff Content ---

  function renderDiffContent(files) {
    diffContent.innerHTML = "";
    if (!files || files.length === 0) {
      return;
    }
    const fragment = document.createDocumentFragment();
    for (const file of files) {
      fragment.appendChild(renderFileSection(file));
    }
    diffContent.appendChild(fragment);
    adjustSplitTableWidths();
  }

  function renderFileSection(file) {
    const section = document.createElement("div");
    section.className = "file-section";
    const path =
      file.status === "deleted" ? file.oldName : file.newName || file.oldName;
    section.id = "file-" + cssId(path);

    // File header
    const header = document.createElement("div");
    header.className = "file-header";

    const displayPath =
      file.status === "renamed"
        ? file.oldName + " \u2192 " + file.newName
        : path;

    var statsHtml = buildChangeStats(file);

    header.innerHTML =
      '<span class="collapse-arrow">&#9660;</span>' +
      '<span class="status-badge ' + file.status + '">' + file.status + "</span>" +
      '<span class="file-path">' + escapeHtml(displayPath) + "</span>" +
      statsHtml;

    header.addEventListener("click", function () {
      section.classList.toggle("collapsed");
    });

    section.appendChild(header);

    // File body
    const body = document.createElement("div");
    body.className = "file-body";
    buildFileBody(file, body);

    section.appendChild(body);
    return section;
  }

  function buildFileBody(file, body) {
    const path =
      file.status === "deleted" ? file.oldName : file.newName || file.oldName;

    if (file.isBinary) {
      body.innerHTML = '<div class="binary-notice">Binary file not shown</div>';
    } else if (file.hunks && file.hunks.length > 0) {
      const table = document.createElement("table");
      table.className = "diff-table " + viewMode;

      if (viewMode === "split") {
        const colgroup = document.createElement("colgroup");
        colgroup.innerHTML =
          '<col style="width: 50px">' +
          '<col style="width: calc(50% - 50px)">' +
          '<col style="width: 1px">' +
          '<col style="width: 50px">' +
          '<col style="width: calc(50% - 50px)">';
        table.appendChild(colgroup);
      }

      const colspan = viewMode === "split" ? 5 : 3;
      const tbody = document.createElement("tbody");

      for (var hi = 0; hi < file.hunks.length; hi++) {
        var hunk = file.hunks[hi];
        // Expand-up row above each hunk
        var expandUpRow = document.createElement("tr");
        expandUpRow.className = "expand-row expand-up";
        var expandUpCell = document.createElement("td");
        expandUpCell.colSpan = colspan;
        expandUpCell.innerHTML =
          '<span class="expand-btn" data-action="expand-up" data-file="' + escapeHtml(path) + '">\u25B2 Expand up</span>' +
          '<span class="expand-btn" data-action="expand-all" data-file="' + escapeHtml(path) + '">\u25A0 Expand all</span>' +
          '<span class="expand-btn" data-action="reset" data-file="' + escapeHtml(path) + '">\u21BA Reset</span>';
        expandUpRow.appendChild(expandUpCell);
        tbody.appendChild(expandUpRow);

        // Hunk content
        if (viewMode === "split") {
          renderHunkSplit(hunk, tbody);
        } else {
          renderHunkUnified(hunk, tbody);
        }

        // Expand-down row below each hunk
        var expandDownRow = document.createElement("tr");
        expandDownRow.className = "expand-row expand-down";
        var expandDownCell = document.createElement("td");
        expandDownCell.colSpan = colspan;
        expandDownCell.innerHTML =
          '<span class="expand-btn" data-action="expand-down" data-file="' + escapeHtml(path) + '">\u25BC Expand down</span>' +
          '<span class="expand-btn" data-action="expand-all" data-file="' + escapeHtml(path) + '">\u25A0 Expand all</span>' +
          '<span class="expand-btn" data-action="reset" data-file="' + escapeHtml(path) + '">\u21BA Reset</span>';
        expandDownRow.appendChild(expandDownCell);
        tbody.appendChild(expandDownRow);
      }

      table.appendChild(tbody);
      body.appendChild(table);

      // Delegate click events on expand buttons
      table.addEventListener("click", function (evt) {
        var btn = evt.target.closest(".expand-btn");
        if (!btn) return;
        evt.stopPropagation();
        var filePath = btn.dataset.file;
        var action = btn.dataset.action;
        if (filePath && action) {
          expandFileContext(filePath, action, evt);
        }
      });
    }
  }

  // --- Hunk Rendering: Split View ---

  function renderHunkSplit(hunk, tbody) {
    // Hunk header row
    const headerRow = document.createElement("tr");
    headerRow.className = "hunk-header";
    const headerCell = document.createElement("td");
    headerCell.colSpan = 5;
    headerCell.textContent = hunk.header;
    headerRow.appendChild(headerCell);
    tbody.appendChild(headerRow);

    // Group lines into pairs for split view
    const lines = hunk.lines;
    let i = 0;

    while (i < lines.length) {
      const line = lines[i];

      if (line.type === "context") {
        // Context: show on both sides
        const tr = document.createElement("tr");
        tr.className = "line-context";
        tr.innerHTML =
          '<td class="line-num old-num">' + (line.oldNum || "") + '</td>' +
          '<td class="line-content old-content">' + escapeHtml(line.content) + '</td>' +
          '<td class="split-divider"></td>' +
          '<td class="line-num new-num">' + (line.newNum || "") + '</td>' +
          '<td class="line-content new-content">' + escapeHtml(line.content) + '</td>';
        tbody.appendChild(tr);
        i++;
      } else if (line.type === "delete") {
        // Collect consecutive deletes
        const deletes = [];
        while (i < lines.length && lines[i].type === "delete") {
          deletes.push(lines[i]);
          i++;
        }
        // Collect consecutive adds after deletes
        const adds = [];
        while (i < lines.length && lines[i].type === "add") {
          adds.push(lines[i]);
          i++;
        }
        // Pair them side by side
        const maxLen = Math.max(deletes.length, adds.length);
        for (let j = 0; j < maxLen; j++) {
          const del = j < deletes.length ? deletes[j] : null;
          const add = j < adds.length ? adds[j] : null;
          const tr = document.createElement("tr");

          if (del && add) {
            // Modification: delete on left, add on right
            tr.innerHTML =
              '<td class="line-num old-num old-del">' + (del.oldNum || "") + '</td>' +
              '<td class="line-content old-content old-del">' + escapeHtml(del.content) + '</td>' +
              '<td class="split-divider"></td>' +
              '<td class="line-num new-num new-add">' + (add.newNum || "") + '</td>' +
              '<td class="line-content new-content new-add">' + escapeHtml(add.content) + '</td>';
          } else if (del) {
            // Delete only
            tr.innerHTML =
              '<td class="line-num old-num old-del">' + (del.oldNum || "") + '</td>' +
              '<td class="line-content old-content old-del">' + escapeHtml(del.content) + '</td>' +
              '<td class="split-divider"></td>' +
              '<td class="line-num new-num empty-cell"></td>' +
              '<td class="line-content new-content empty-cell"></td>';
          } else {
            // Add only
            tr.innerHTML =
              '<td class="line-num old-num empty-cell"></td>' +
              '<td class="line-content old-content empty-cell"></td>' +
              '<td class="split-divider"></td>' +
              '<td class="line-num new-num new-add">' + (add.newNum || "") + '</td>' +
              '<td class="line-content new-content new-add">' + escapeHtml(add.content) + '</td>';
          }
          tbody.appendChild(tr);
        }
      } else if (line.type === "add") {
        // Standalone add (not preceded by delete)
        const tr = document.createElement("tr");
        tr.innerHTML =
          '<td class="line-num old-num empty-cell"></td>' +
          '<td class="line-content old-content empty-cell"></td>' +
          '<td class="split-divider"></td>' +
          '<td class="line-num new-num new-add">' + (line.newNum || "") + '</td>' +
          '<td class="line-content new-content new-add">' + escapeHtml(line.content) + '</td>';
        tbody.appendChild(tr);
        i++;
      } else {
        i++;
      }
    }
  }

  // --- Hunk Rendering: Unified View ---

  function renderHunkUnified(hunk, tbody) {
    // Hunk header row
    const headerRow = document.createElement("tr");
    headerRow.className = "hunk-header";
    const headerCell = document.createElement("td");
    headerCell.colSpan = 3;
    headerCell.textContent = hunk.header;
    headerRow.appendChild(headerCell);
    tbody.appendChild(headerRow);

    for (const line of hunk.lines) {
      const tr = document.createElement("tr");

      if (line.type === "context") {
        tr.className = "line-context";
        tr.innerHTML =
          '<td class="line-num">' + (line.oldNum || "") + '</td>' +
          '<td class="line-num">' + (line.newNum || "") + '</td>' +
          '<td class="line-content">' + escapeHtml(line.content) + '</td>';
      } else if (line.type === "add") {
        tr.className = "line-add";
        tr.innerHTML =
          '<td class="line-num"></td>' +
          '<td class="line-num">' + (line.newNum || "") + '</td>' +
          '<td class="line-content">' + escapeHtml(line.content) + '</td>';
      } else if (line.type === "delete") {
        tr.className = "line-delete";
        tr.innerHTML =
          '<td class="line-num">' + (line.oldNum || "") + '</td>' +
          '<td class="line-num"></td>' +
          '<td class="line-content">' + escapeHtml(line.content) + '</td>';
      }

      tbody.appendChild(tr);
    }
  }

  // --- Interactions ---

  function toggleViewMode(mode) {
    if (mode === viewMode) return;
    viewMode = mode;

    btnSplit.classList.toggle("active", mode === "split");
    btnUnified.classList.toggle("active", mode === "unified");

    // Re-render diffs only (keep file tree as-is)
    renderDiffContent(currentFiles);
  }

  async function loadDiff() {
    showLoading();
    // Reset expansion levels on a fresh diff load
    fileContextLevels = {};
    try {
      const base = basePicker.value || undefined;
      const target = targetPicker.value || undefined;
      const data = await fetchDiff(base, target);
      currentFiles = data.files || [];
      renderFileTree(currentFiles);
      renderDiffContent(currentFiles);
    } catch (err) {
      showError("Failed to load diff: " + err.message);
    }
  }

  function scrollToFile(filename) {
    const id = "file-" + cssId(filename);
    const el = document.getElementById(id);
    if (el) {
      // Ensure the section is expanded
      el.classList.remove("collapsed");
      el.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  }

  // --- Helpers ---

  function adjustSplitTableWidths() {
    const tables = diffContent.querySelectorAll(".diff-table.split");
    for (const table of tables) {
      // Reset any previous adjustment
      table.style.minWidth = "";

      // Measure the widest content on each side using a temporary
      // off-screen element to get the natural (unwrapped) width.
      const probe = document.createElement("span");
      probe.style.cssText =
        "position:absolute;visibility:hidden;white-space:pre;" +
        "font-family:var(--font-mono);font-size:var(--font-size);padding:0 10px";
      document.body.appendChild(probe);

      let maxWidth = 0;
      const cells = table.querySelectorAll(".line-content");
      for (const cell of cells) {
        if (cell.classList.contains("empty-cell")) continue;
        probe.textContent = cell.textContent;
        if (probe.offsetWidth > maxWidth) {
          maxWidth = probe.offsetWidth;
        }
      }
      document.body.removeChild(probe);

      // Each side needs: lineNum (50px) + content (maxWidth)
      // Total: 2 * (50 + maxWidth) + 1px divider
      const needed = 2 * (50 + maxWidth) + 1;
      const container = table.closest(".file-body");
      if (container && needed > container.clientWidth) {
        table.style.minWidth = needed + "px";
      }
    }
  }

  function escapeHtml(str) {
    if (!str) return "";
    return str
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function cssId(path) {
    return path.replace(/[^a-zA-Z0-9_-]/g, "-");
  }

  function showLoading() {
    diffContent.innerHTML = '<div class="loading">Loading diff...</div>';
    fileTreeContent.innerHTML = '<div class="loading">Loading...</div>';
  }

  function showError(message) {
    diffContent.innerHTML = '<div class="error-message">' + escapeHtml(message) + "</div>";
  }

  // --- Commit Picker ---

  async function populateCommits() {
    try {
      const commits = await fetchCommits();

      // --- Base picker ---
      basePicker.innerHTML = "";
      const baseDefault = document.createElement("option");
      baseDefault.value = "";
      baseDefault.textContent = "Merge base";
      basePicker.appendChild(baseDefault);

      // --- Target picker (target-picker) ---
      targetPicker.innerHTML = "";
      const targetDefault = document.createElement("option");
      targetDefault.value = "";
      targetDefault.textContent = "Working tree";
      targetPicker.appendChild(targetDefault);

      if (commits && commits.length > 0) {
        for (const c of commits) {
          const shortHash = c.hash.substring(0, 7);
          const msg =
            c.message.length > 60
              ? c.message.substring(0, 57) + "..."
              : c.message;
          const label = shortHash + " " + msg;

          const baseOpt = document.createElement("option");
          baseOpt.value = c.hash;
          baseOpt.textContent = label;
          basePicker.appendChild(baseOpt);

          const targetOpt = document.createElement("option");
          targetOpt.value = c.hash;
          targetOpt.textContent = label;
          targetPicker.appendChild(targetOpt);
        }
      }
    } catch (err) {
      basePicker.innerHTML = "";
      const opt = document.createElement("option");
      opt.value = "";
      opt.textContent = "Failed to load commits";
      basePicker.appendChild(opt);

      targetPicker.innerHTML = "";
      const topt = document.createElement("option");
      topt.value = "";
      topt.textContent = "Failed to load commits";
      targetPicker.appendChild(topt);
    }
  }

  // --- Event Listeners ---

  btnSplit.addEventListener("click", function () { toggleViewMode("split"); });
  btnUnified.addEventListener("click", function () { toggleViewMode("unified"); });

  for (const picker of [basePicker, targetPicker]) {
    picker.addEventListener("change", loadDiff);
  }

  // --- Init ---

  async function init() {
    showLoading();

    // Fetch commits and diff in parallel
    const results = await Promise.allSettled([
      populateCommits(),
      fetchDiff(),
    ]);
    const diffResult = results[1];

    if (diffResult.status === "fulfilled") {
      currentFiles = diffResult.value.files || [];
      renderFileTree(currentFiles);
      renderDiffContent(currentFiles);
    } else {
      showError("Failed to load diff: " + (diffResult.reason?.message || "Unknown error"));
    }
  }

  init();
})();