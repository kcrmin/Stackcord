const test = require("node:test");
const assert = require("node:assert/strict");
const vm = require("node:vm");
const fs = require("node:fs");
const path = require("node:path");
// Execute the shipped application with a minimal DOM and controlled network.
// This exercises the project's refresh state machine, not a mock component.
class Element {
  constructor() {
    this.children = [];
    this.value = "";
    this.checked = false;
    this.hidden = false;
    this.elements = { project: { value: "" } };
    this._text = "";
  }
  set textContent(v) {
    this._text = String(v);
    this.children = [];
  }
  get textContent() {
    return this._text + this.children.map((c) => c.textContent).join(" ");
  }
  append(...nodes) {
    this.children.push(...nodes);
    for (const n of nodes) n.parent = this;
  }
  replaceChildren(...nodes) {
    this._text = "";
    this.children = [];
    this.append(...nodes);
  }
  get firstChild() {
    return this.children[0];
  }
  remove() {
    this.parent.children = this.parent.children.filter((c) => c !== this);
  }
  reset() {}
  querySelector() {
    return new Element();
  }
}
function app(fetch) {
  const elements = new Map();
  const element = (id) => {
    if (!elements.has(id)) elements.set(id, new Element());
    return elements.get(id);
  };
  element("budget").value = "16384";
  const document = {
    getElementById: element,
    createElement: () => new Element(),
    querySelectorAll: () => [],
    documentElement: {},
    hidden: false,
  };
  const ctx = vm.createContext({
    document,
    navigator: { language: "en" },
    fetch,
    URLSearchParams,
    crypto: require("node:crypto"),
    setInterval: () => {},
    queueMicrotask,
    console,
  });
  vm.runInContext(
    fs.readFileSync(path.join(__dirname, "assets/app.js"), "utf8"),
    ctx,
  );
  return { run: (code) => vm.runInContext(code, ctx), element };
}
const response = (value) => ({ ok: true, json: async () => value });
test("switching project during refresh never leaves old task cards", async () => {
  let hold = false,
    release;
  const gate = new Promise((resolve) => (release = resolve));
  const { run, element } = app(async (url) => {
    if (url === "/v1/projects") {
      if (hold) {
        hold = false;
        await gate;
      }
      return response(["alpha", "beta"]);
    }
    if (url.startsWith("/v1/tasks")) {
      const p = new URL(url, "http://localhost").searchParams.get("project");
      return response([
        {
          id: "same-id",
          definition: { goal: p.toUpperCase() + " TASK" },
          state: "ready",
        },
      ]);
    }
    if (url.startsWith("/v1/events"))
      return response({ events: [], next: 0, more: false });
    throw Error(url);
  });
  await run("token='credential';project='alpha';refresh()");
  assert.match(element("tasks").textContent, /ALPHA TASK/);
  hold = true;
  const old = run("refresh()");
  element("project").value = "beta";
  run("$('project').onchange()");
  assert.doesNotMatch(element("tasks").textContent, /ALPHA TASK/);
  release();
  await old;
  await new Promise(setImmediate);
  assert.equal(run("project"), "beta");
  assert.match(element("tasks").textContent, /BETA TASK/);
});
test("disconnect discards in-flight private project data", async () => {
  let release;
  const gate = new Promise((resolve) => (release = resolve));
  const { run, element } = app(async () => {
    await gate;
    return response(["private-project"]);
  });
  const inFlight = run("token='credential';refresh()");
  run("$('disconnect').onclick()");
  release();
  await inFlight;
  assert.equal(element("workspace").hidden, true);
  assert.equal(element("project").children.length, 0);
  assert.equal(run("token"), "");
});

test("automatic refresh preserves a new-project draft", async () => {
  const { run, element } = app(async (url) => {
    if (url === "/v1/projects") return response(["existing"]);
    if (url.startsWith("/v1/tasks")) return response([]);
    if (url.startsWith("/v1/events"))
      return response({ events: [], next: 0, more: false });
    throw Error(url);
  });
  await run("token='credential';project='existing';refresh()");
  element("create-form").elements.project.value = "new-project";
  await run("refresh()");
  assert.equal(element("create-form").elements.project.value, "new-project");
});
