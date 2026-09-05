/**
 * Docs chrome: sidebar toggle and title search against index.json.
 */
(function () {
  const menu = document.getElementById("menu-toggle");
  const side = document.getElementById("sidebar");
  if (menu && side) {
    menu.addEventListener("click", () => side.classList.toggle("open"));
  }

  const input = document.getElementById("q");
  const hits = document.getElementById("hits");
  if (!input || !hits) return;

  let pages = [];
  fetch(input.dataset.index || "index.json")
    .then((r) => r.json())
    .then((data) => {
      pages = Array.isArray(data) ? data : [];
    })
    .catch(() => {
      pages = [];
    });

  /** Render matching titles into the hits list. */
  function show(q) {
    const needle = q.trim().toLowerCase();
    hits.replaceChildren();
    if (needle.length < 2) {
      hits.hidden = true;
      return;
    }
    const found = pages
      .filter((p) =>
        String(p.title || "").toLowerCase().includes(needle) ||
        String(p.text || "").toLowerCase().includes(needle),
      )
      .slice(0, 12);
    for (const p of found) {
      const li = document.createElement("li");
      const a = document.createElement("a");
      a.href = p.permalink;
      a.textContent = p.title;
      li.appendChild(a);
      hits.appendChild(li);
    }
    hits.hidden = found.length === 0;
  }

  input.addEventListener("input", () => show(input.value));
  input.addEventListener("blur", () => {
    window.setTimeout(() => {
      hits.hidden = true;
    }, 150);
  });
})();
