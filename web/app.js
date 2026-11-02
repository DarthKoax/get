const statusEl = document.getElementById("status");
let statusTimer = null;
function showStatus(msg, isError = false) {
  statusEl.textContent = msg;
  statusEl.classList.remove("status-error", "status-success");
  statusEl.classList.add(isError ? "status-error" : "status-success");
  if (statusTimer) clearTimeout(statusTimer);
  statusTimer = setTimeout(() => {
    statusEl.textContent = "";
    statusEl.classList.remove("status-error", "status-success");
  }, isError ? 5000 : 3000);
}

const loadingEl = document.getElementById("loading");
const resolveInput = document.getElementById("resolve-input");
const resolveBtn = document.getElementById("resolve-btn");
const nameInput = document.getElementById("name");
const urlInput = document.getElementById("url");
const searchUrlInput = document.getElementById("search-url");
const descInput = document.getElementById("description");
const addBtn = document.getElementById("add-btn");
const addForm = document.getElementById("add-form");
const filterInput = document.getElementById("filter-input");
const listEl = document.getElementById("shortcuts-list");
const downloadBtn = document.getElementById("download-btn");
const hamburgerBtn = document.getElementById("hamburger-btn");
const menuPanel = document.getElementById("menu-panel");

function setEnabled(enabled) {
  resolveInput.disabled = !enabled;
  resolveBtn.disabled = !enabled;
  nameInput.disabled = !enabled;
  urlInput.disabled = !enabled;
  searchUrlInput.disabled = !enabled;
  descInput.disabled = !enabled;
  addBtn.disabled = !enabled;
  filterInput.disabled = !enabled;
}

setEnabled(false);

function waitForGo() {
  return new Promise((resolve, reject) => {
    const timeout = setTimeout(() => reject(new Error("WASM initialization timed out")), 10000);
    const check = () => {
      if (typeof goReady === "function") {
        goReady((res) => {
          if (res && res.error) {
            showStatus(res.error, true);
            setTimeout(check, 200);
          } else {
            clearTimeout(timeout);
            resolve();
          }
        });
      } else {
        setTimeout(check, 100);
      }
    };
    check();
  });
}

async function loadWasm() {
  const go = new Go();
  const resp = await fetch("main.wasm");
  const result = await WebAssembly.instantiateStreaming(resp, go.importObject);
  go.run(result.instance);
}

let mode = "static";
let api = null;
let currentFilter = "";

async function detectMode() {
  const meta = document.querySelector('meta[name="app-mode"]');
  if (meta && meta.content && meta.content !== "auto") {
    return meta.content;
  }
  try {
    const res = await fetch("/api/shortcuts");
    if (res.ok) return "server";
  } catch (e) {
    console.log("Mode detection: API not available, using static mode", e);
  }
  return "static";
}

async function init() {
  mode = await detectMode();
  document.body.dataset.mode = mode;

  const modeBadge = document.getElementById("mode-badge");
  if (modeBadge) {
    modeBadge.textContent = mode === "server" ? "Server Mode" : "Static Mode";
  }

  if (mode === "server") {
    api = (window.apiClient) ? window.apiClient : new APIClient();
    setEnabled(true);
    
    // Setup hamburger menu toggle
    if (hamburgerBtn && menuPanel) {
      hamburgerBtn.addEventListener("click", () => {
        const isExpanded = hamburgerBtn.getAttribute("aria-expanded") === "true";
        hamburgerBtn.setAttribute("aria-expanded", String(!isExpanded));
        menuPanel.classList.toggle("active");
      });
    }
  } else {
    try {
      await loadWasm();
    } catch (err) {
      console.error("WASM load error:", err);
      showStatus("Failed to load WASM: " + err, true);
    }
    try {
      await waitForGo();
    } catch (err) {
      console.error("WASM init error:", err);
      showStatus(String(err), true);
    }
    setEnabled(true);
  }

  // Check for ?get=<name> query parameter
  const urlParams = new URLSearchParams(window.location.search);
  const getParam = urlParams.get("get");
  if (getParam) {
    resolveByName(getParam);
  } else {
    loadShortcuts();
  }
}

init();

function loadShortcuts() {
  if (mode === "server") {
    (async () => {
      if (loadingEl) loadingEl.style.display = "block";
      try {
        let data;
        if (currentFilter && currentFilter.trim() !== "") {
          data = await api.searchShortcuts(currentFilter);
        } else {
          data = await api.listShortcuts();
        }
        renderShortcuts(Array.isArray(data) ? data : []);
      } catch (e) {
        showStatus("Failed to load shortcuts: " + e, true);
      } finally {
        if (loadingEl) loadingEl.style.display = "none";
      }
    })();
    return;
  }
  if (typeof goSearchShortcuts !== "function") return;
  goSearchShortcuts(currentFilter, (res) => {
    if (res && res.error) {
      showStatus(res.error, true);
      return;
    }
    renderShortcuts(Array.isArray(res) ? res : []);
  });
}

function navigateToURL(url) {
  // Check for browser-restricted protocols
  const restrictedProtocols = ['chrome:', 'edge:', 'about:', 'moz-extension:', 'chrome-extension:'];
  const isRestricted = restrictedProtocols.some(protocol => url.startsWith(protocol));
  
  if (isRestricted) {
    showStatus(`Browser security prevents direct navigation to ${url}. Please copy and paste this URL into your address bar.`, true);
    // Try to copy to clipboard
    if (navigator.clipboard) {
      navigator.clipboard.writeText(url).then(() => {
        showStatus(`URL copied to clipboard: ${url}. Paste it in your address bar.`, false);
      }).catch(() => {
        showStatus(`Cannot auto-navigate to ${url}. Please copy manually.`, true);
      });
    }
    return false;
  }
  
  window.location.href = url;
  return true;
}

function renderShortcuts(shortcuts) {
  if (loadingEl) loadingEl.style.display = "none";
  listEl.innerHTML = "";
  if (!shortcuts || shortcuts.length === 0) {
    const p = document.createElement("p");
    p.className = "empty";
    p.textContent = currentFilter
      ? `No shortcuts match "${currentFilter}".`
      : "No shortcuts yet.";
    listEl.appendChild(p);
    return;
  }
  shortcuts.forEach((s) => {
    const card = document.createElement("div");
    card.className = "shortcut-card";
    card.dataset.id = s.id;

    const view = document.createElement("div");
    view.className = "shortcut-view";
    view.style.cursor = "pointer";
    view.addEventListener("click", (e) => {
      // Don't navigate if clicking buttons
      if (e.target.closest("button")) return;
      navigateToURL(s.url);
    });
    const nameEl = document.createElement("div");
    nameEl.className = "name";
    nameEl.textContent = s.name;
    const urlEl = document.createElement("div");
    urlEl.className = "url";
    urlEl.textContent = s.url;
    const descEl = document.createElement("div");
    descEl.className = "desc";
    descEl.textContent = s.description || "";
    const metaEl = document.createElement("div");
    metaEl.className = "meta";
    metaEl.textContent = s.created_at;
    const actions = document.createElement("div");
    actions.className = "actions";
    if (mode === "server") {
      const editBtn = document.createElement("button");
      editBtn.innerHTML = `<svg width="12" height="12" viewBox="0 0 12 12" style="vertical-align: middle; margin-right: 4px;"><rect x="1" y="9" width="2" height="2" fill="currentColor"/><rect x="3" y="7" width="2" height="2" fill="currentColor"/><rect x="5" y="5" width="2" height="2" fill="currentColor"/><rect x="7" y="3" width="2" height="2" fill="currentColor"/><rect x="9" y="1" width="2" height="2" fill="currentColor"/></svg>Edit`;
      editBtn.addEventListener("click", () => enterEdit(card, s));
      const delBtn = document.createElement("button");
      delBtn.className = "btn-danger";
      delBtn.innerHTML = `<svg width="12" height="12" viewBox="0 0 12 12" style="vertical-align: middle; margin-right: 4px;"><rect x="2" y="3" width="8" height="1" fill="currentColor"/><rect x="4" y="1" width="4" height="2" fill="currentColor"/><rect x="3" y="4" width="6" height="7" fill="currentColor"/><rect x="5" y="6" width="1" height="3" fill="#1a1a1a"/><rect x="7" y="6" width="1" height="3" fill="#1a1a1a"/></svg>Delete`;
      delBtn.setAttribute("aria-label", "Delete " + s.name);
      delBtn.addEventListener("click", () => handleDelete(s.id, s.name));
      actions.append(editBtn, delBtn);
    }
    view.append(nameEl, urlEl);
    if (s.search_url) {
      const searchBadge = document.createElement("span");
      searchBadge.className = "search-badge";
      searchBadge.textContent = "Search";
      searchBadge.title = "Search URL: " + s.search_url;
      view.append(searchBadge);
    }
    if (s.description) view.append(descEl);
    view.append(metaEl, actions);

    const editForm = document.createElement("div");
    editForm.className = "shortcut-edit";
    editForm.style.display = "none";
    const nameInputEdit = document.createElement("input");
    nameInputEdit.value = s.name;
    nameInputEdit.setAttribute("aria-label", "Name");
    nameInputEdit.required = true;
    nameInputEdit.pattern = "^[a-z0-9]([a-z0-9-]*[a-z0-9])?$";
    const urlInputEdit = document.createElement("input");
    urlInputEdit.type = "text";
    urlInputEdit.value = s.url;
    urlInputEdit.setAttribute("aria-label", "URL");
    urlInputEdit.required = true;
    const searchUrlInputEdit = document.createElement("input");
    searchUrlInputEdit.type = "text";
    searchUrlInputEdit.value = s.search_url || "";
    searchUrlInputEdit.placeholder = "Optional search URL";
    searchUrlInputEdit.setAttribute("aria-label", "Search URL");
    const descInputEdit = document.createElement("input");
    descInputEdit.value = s.description || "";
    descInputEdit.setAttribute("aria-label", "Description");
    const saveBtn = document.createElement("button");
    saveBtn.innerHTML = `<svg width="12" height="12" viewBox="0 0 12 12" style="vertical-align: middle; margin-right: 4px;"><polygon points="2,6 5,9 10,3" fill="none" stroke="currentColor" stroke-width="2"/></svg>Save`;
    saveBtn.addEventListener("click", () => handleUpdate(s.id, nameInputEdit.value, urlInputEdit.value, searchUrlInputEdit.value, descInputEdit.value));
    const cancelBtn = document.createElement("button");
    cancelBtn.innerHTML = `<svg width="12" height="12" viewBox="0 0 12 12" style="vertical-align: middle; margin-right: 4px;"><rect x="2" y="2" width="2" height="2" fill="currentColor"/><rect x="4" y="4" width="2" height="2" fill="currentColor"/><rect x="6" y="6" width="2" height="2" fill="currentColor"/><rect x="8" y="8" width="2" height="2" fill="currentColor"/><rect x="8" y="2" width="2" height="2" fill="currentColor"/><rect x="6" y="4" width="2" height="2" fill="currentColor"/><rect x="4" y="6" width="2" height="2" fill="currentColor"/><rect x="2" y="8" width="2" height="2" fill="currentColor"/></svg>Cancel`;
    cancelBtn.addEventListener("click", () => {
      view.style.display = "";
      editForm.style.display = "none";
    });
    editForm.append(nameInputEdit, urlInputEdit, searchUrlInputEdit, descInputEdit, saveBtn, cancelBtn);

    editForm.addEventListener("keydown", (e) => {
      if (e.key === "Enter" && e.target.tagName === "INPUT") {
        e.preventDefault();
        handleUpdate(s.id, nameInputEdit.value, urlInputEdit.value, searchUrlInputEdit.value, descInputEdit.value);
      }
      if (e.key === "Escape") {
        view.style.display = "";
        editForm.style.display = "none";
      }
    });

    card.append(view, editForm);
    listEl.appendChild(card);
  });
}

function enterEdit(card, s) {
  const view = card.querySelector(".shortcut-view");
  const editForm = card.querySelector(".shortcut-edit");
  view.style.display = "none";
  editForm.style.display = "flex";
  const first = editForm.querySelector("input");
  if (first) first.focus();
}

function handleUpdate(id, name, url, searchUrl, desc) {
  if (mode === "server") {
    api.updateShortcut(id, { name: name.trim(), url: url.trim(), search_url: searchUrl.trim(), description: desc }).then((res) => {
      if (res && res.error) {
        showStatus(res.error, true);
        return;
      }
      showStatus("Shortcut updated");
      loadShortcuts();
    }).catch((err) => showStatus(String(err), true));
    return;
  }
  if (typeof goUpdateShortcut !== "function") return;
  goUpdateShortcut(id, name.trim(), url.trim(), searchUrl.trim(), desc, (res) => {
    if (res && res.error) {
      showStatus(res.error, true);
      return;
    }
    showStatus("Shortcut updated");
    loadShortcuts();
  });
}

function handleDelete(id, name) {
  if (!confirm("Delete shortcut \"" + name + "\"?")) return;
  if (mode === "server") {
    api.deleteShortcut(id).then((res) => {
      if (res && res.error) {
        showStatus(res.error, true);
        return;
      }
      showStatus("Shortcut deleted");
      loadShortcuts();
    }).catch((err) => showStatus(String(err), true));
    return;
  }
  if (typeof goDeleteShortcut !== "function") return;
  goDeleteShortcut(id, (res) => {
    if (res && res.error) {
      showStatus(res.error, true);
      return;
    }
    showStatus("Shortcut deleted");
    loadShortcuts();
  });
}

function resolveByName(name) {
  const trimmedName = name.trim();
  if (!trimmedName) return;
  if (mode === "server") {
    api.resolveShortcut(trimmedName).then((res) => {
      if (res && res.error) {
        showStatus(res.error, true);
        loadShortcuts();
        return;
      }
      if (res && res.url) {
        const newUrl = new URL(window.location);
        newUrl.searchParams.set("get", trimmedName);
        window.history.pushState({}, "", newUrl);
        navigateToURL(res.url);
      }
    }).catch((err) => showStatus(String(err), true));
    return;
  }
  if (typeof goResolveShortcut !== "function") {
    showStatus("WASM not ready yet", true);
    return;
  }
  goResolveShortcut(trimmedName, (res) => {
    if (res && res.error) {
      showStatus(res.error, true);
      loadShortcuts();
      return;
    }
    if (res && res.url) {
      const newUrl = new URL(window.location);
      newUrl.searchParams.set("get", trimmedName);
      window.history.pushState({}, "", newUrl);
      navigateToURL(res.url);
    }
  });
}

function doResolve() {
  const name = resolveInput.value.trim();
  if (!name) return;
  resolveByName(name);
}
resolveBtn.addEventListener("click", doResolve);
resolveInput.addEventListener("keydown", (e) => {
  if (e.key === "Enter") {
    e.preventDefault();
    doResolve();
  }
});

function validateField(input, label) {
  if (input.validity && !input.validity.valid) {
    let msg = label + " is invalid";
    if (input.validity.valueMissing) msg = label + " is required";
    else if (input.validity.patternMismatch) msg = label + " format is invalid";
    else if (input.validity.typeMismatch) msg = label + " must be a valid URL";
    showStatus(msg, true);
    return false;
  }
  return true;
}
[nameInput, urlInput].forEach((inp) => {
  inp.addEventListener("blur", () => validateField(inp, inp === nameInput ? "Name" : "URL"));
});

addForm.addEventListener("submit", (e) => {
  e.preventDefault();
  const name = nameInput.value.trim();
  const url = urlInput.value.trim();
  const searchUrl = searchUrlInput.value.trim();
  const desc = descInput.value;
  if (!name) {
    showStatus("Name is required", true);
    return;
  }
  if (!url) {
    showStatus("URL is required", true);
    return;
  }
  if (mode === "server") {
    api.createShortcut({ name, url, search_url: searchUrl, description: desc }).then((res) => {
      if (res && res.error) {
        showStatus(res.error, true);
        return;
      }
      nameInput.value = "";
      urlInput.value = "";
      searchUrlInput.value = "";
      descInput.value = "";
      showStatus("Shortcut added");
      loadShortcuts();
    }).catch((err) => showStatus(String(err), true));
    return;
  }
  if (typeof goCreateShortcut !== "function") {
    showStatus("WASM not ready yet", true);
    return;
  }
  goCreateShortcut(name, url, searchUrl, desc, (res) => {
    if (res && res.error) {
      showStatus(res.error, true);
      return;
    }
    nameInput.value = "";
    urlInput.value = "";
    searchUrlInput.value = "";
    descInput.value = "";
    showStatus("Shortcut added");
    loadShortcuts();
  });
});

window.addEventListener("popstate", () => {
  const urlParams = new URLSearchParams(window.location.search);
  const getParam = urlParams.get("get");
  if (getParam) {
    resolveByName(getParam);
  }
});

filterInput.addEventListener("input", () => {
  currentFilter = filterInput.value;
  loadShortcuts();
});

if (downloadBtn) {
  downloadBtn.addEventListener("click", async () => {
    if (mode === "server" && api) {
      const originalText = downloadBtn.textContent;
      downloadBtn.disabled = true;
      downloadBtn.textContent = "Downloading…";
      try {
        await api.downloadDatabase();
        showStatus("Database downloaded", false);
      } catch (err) {
        showStatus(String(err), true);
      } finally {
        downloadBtn.disabled = false;
        downloadBtn.textContent = originalText;
      }
    }
  });
}
