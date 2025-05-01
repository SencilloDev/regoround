let editors = {};

function syncEditorHeights() {
  const ids = ["input", "data", "response"];
  const editors = ids.map(id => document.getElementById(id));

  editors.forEach((editor, _, all) => {
    editor.addEventListener("input", () => {
      const height = editor.offsetHeight;
      all.forEach((e) => {
        if (e !== editor) e.style.height = `${height}px`;
      });
    });
  });
}

// Call it after DOM is ready
document.addEventListener("DOMContentLoaded", syncEditorHeights);


function initializeEditors() {
  // Initialize Rego editor
  editors.package = CodeMirror.fromTextArea(document.getElementById("package"), {
    mode: "rego",
    lineNumbers: true,
    theme: "default",
    autoCloseBrackets: true,
    matchBrackets: true,
    lineWrapping: true,
  });

  // Initialize JSON editors
  editors.input = CodeMirror.fromTextArea(document.getElementById("input"), {
    mode: "application/json",
    lineNumbers: true,
 theme: "default",
    autoCloseBrackets: true,
    matchBrackets: true,
  });

  editors.data = CodeMirror.fromTextArea(document.getElementById("data"), {
    mode: "application/json",
    lineNumbers: true,
    theme: "default",
    autoCloseBrackets: true,
    matchBrackets: true,
  });

  Object.values(editors).forEach((editor) => {
    let height = (editor.options.mode === 'rego') ? "90%" : "200px";
    editor.setSize("100%", height);
  });
}


function saveEditorContent() {
  Object.entries(editors).forEach(([id, editor]) => {
    editor.save();
  });
}

function formatJSON() {
  try {
    ["input", "data"].forEach((id) => {
      const editor = editors[id];
      const content = editor.getValue().trim();

      if (content) {
        const formatted = JSON.stringify(JSON.parse(content), null, 2);
        editor.setValue(formatted);
      }
    });
  } catch (e) {
    alert("Invalid JSON in input or data field");
  }
}

function saveEditorContent() {
  Object.entries(editors).forEach(([id, editor]) => {
    editor.save();
  });
}

function hydrate() {
  const parser = new URL(window.location.href);
  let m = new Map(Object.entries(editors));
  parser.searchParams.forEach((val, param) => {
    if (editors[param]) {
      editors[param].setValue(decompressData(val))
    }
  });
}

function updateSearchParams(params) {
  const url = new URL(window.location.href);
  Object.entries(params).forEach(([key, value]) => {
    url.searchParams.set(key, value);
  })

  history.replaceState(null, '', url);
}

function compressAndUpdateURL() {
  const params = {};
  Object.entries(editors).forEach(([id, editor]) => {
    params[id] = compressData(editor.getValue());
  })
  updateSearchParams(params);
}




// Initialize when the DOM is ready
function init() {
  initializeEditors();
  hydrate();

  // Add format button handler
  document.getElementById("format").addEventListener("click", formatJSON);

  // Add form submit handler
  document.getElementById("evaluate").addEventListener("click", async (e) => {
      e.preventDefault();

      saveEditorContent();
      compressAndUpdateURL()

       const payload = {
         input: editors.input.getValue(),
         data: editors.data.getValue(),
         package: editors.package.getValue()
       };

      try {
        const response = await fetch("/api/v1/evaluate", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(payload),
        });

        const text = await response.text();
        document.getElementById("response").value = text;
      } catch (err) {
        document.getElementById("response").value = "Error: " + err.message;
      }
    });
};
