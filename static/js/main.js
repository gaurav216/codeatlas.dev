(() => {
  const ready = (callback) => {
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", callback, { once: true });
      return;
    }
    callback();
  };

  ready(() => {
    document.querySelectorAll('form[role="search"]').forEach((form) => {
      form.addEventListener("submit", (event) => {
        const input = form.querySelector('input[type="search"]');
        if (!input) return;

        input.value = input.value.trim();
        if (input.value === "") {
          event.preventDefault();
          input.focus();
        }
      });
    });

    renderMermaidBlocks();
    addCopyButtons();
    buildArticleToc();
  });

  function renderMermaidBlocks() {
    const signatures = [
      "graph ",
      "flowchart ",
      "sequenceDiagram",
      "classDiagram",
      "stateDiagram",
      "erDiagram",
      "journey",
      "gantt",
      "pie ",
      "mindmap",
      "timeline"
    ];

    document.querySelectorAll("#article-content pre").forEach((pre) => {
      const code = pre.querySelector("code");
      if (!code) return;

      const text = code.textContent.trim();
      const className = code.className || pre.className || "";
      const isMermaid = className.includes("language-mermaid") || signatures.some((prefix) => text.startsWith(prefix));
      if (!isMermaid) return;

      const diagram = document.createElement("div");
      diagram.className = "mermaid";
      diagram.textContent = text;
      pre.replaceWith(diagram);
    });

    if (window.mermaid) {
      window.mermaid.initialize({
        startOnLoad: false,
        theme: "default",
        securityLevel: "strict"
      });
      window.mermaid.run({ querySelector: ".mermaid" });
    }
  }

  function addCopyButtons() {
    document.querySelectorAll("#article-content pre").forEach((pre) => {
      if (pre.closest(".code-shell")) return;

      const wrapper = document.createElement("div");
      wrapper.className = "code-shell";
      pre.parentNode.insertBefore(wrapper, pre);
      wrapper.appendChild(pre);

      const button = document.createElement("button");
      button.className = "copy-code";
      button.type = "button";
      button.textContent = "Copy";
      wrapper.appendChild(button);

      button.addEventListener("click", async () => {
        const code = pre.textContent;
        try {
          await navigator.clipboard.writeText(code);
          button.textContent = "Copied";
        } catch {
          fallbackCopy(code);
          button.textContent = "Copied";
        }
        window.setTimeout(() => {
          button.textContent = "Copy";
        }, 1400);
      });
    });
  }

  function buildArticleToc() {
    const content = document.querySelector("#article-content");
    const toc = document.querySelector("#article-toc");
    if (!content || !toc) return;

    const headings = Array.from(content.querySelectorAll("h2, h3"));
    if (headings.length === 0) {
      toc.innerHTML = '<span class="text-slate-500">No sections yet.</span>';
      return;
    }

    headings.forEach((heading) => {
      if (!heading.id) {
        heading.id = slugify(heading.textContent);
      }

      const link = document.createElement("a");
      link.className = heading.tagName === "H3" ? "toc-link toc-link--h3" : "toc-link";
      link.href = `#${heading.id}`;
      link.textContent = heading.textContent;
      toc.appendChild(link);
    });
  }

  function slugify(value) {
    return value
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "");
  }

  function fallbackCopy(text) {
    const area = document.createElement("textarea");
    area.value = text;
    area.setAttribute("readonly", "");
    area.style.position = "fixed";
    area.style.opacity = "0";
    document.body.appendChild(area);
    area.select();
    document.execCommand("copy");
    area.remove();
  }
})();
