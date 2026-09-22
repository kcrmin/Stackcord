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
    this.elements = { project: { value: "", focus() {} } };
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
  reset() {
    this.fields = {};
  }
  setAttribute(k, v) {
    this[k] = v;
  }
  focus() {}
  showModal() {
    this.open = true;
  }
  close() {
    this.open = false;
  }
  querySelector() {
    return new Element();
  }
}
function app(fetch, { hash = "" } = {}) {
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
    location: { hash, origin: "http://127.0.0.1:7331", pathname: "/" },
    history: { replaceState() {} },
    FormData: class {
      constructor(f) {
        this.fields = f.fields || {};
      }
      get(k) {
        return this.fields[k] || "";
      }
    },
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

test("first task needs only a goal and creates stable identifiers before a network retry", async () => {
  const attempts = [];
  const { run, element } = app(async (url, opts) => {
    if (url === "/v1/apply") {
      attempts.push(JSON.parse(opts.body));
      throw Error("connection interrupted");
    }
    throw Error(url);
  });
  element("create-form").fields = {
    project: "",
    task: "",
    goal: "예약 API 만들기",
    constraints: "",
  };
  const submit = () =>
    run(
      "$('create-form').onsubmit({preventDefault(){},currentTarget:$('create-form')})",
    );
  await submit();
  await submit();
  assert.equal(attempts.length, 2);
  assert.equal(attempts[0].project, "my-project");
  assert.match(attempts[0].task, /^task-/);
  assert.deepEqual(attempts[1], attempts[0]);
  assert.equal(attempts[0].definition.goal, "예약 API 만들기");
});

test("completed tasks are out of the active view but remain available in All", async () => {
  const { run, element } = app(async (url) => {
    if (url === "/v1/projects") return response(["p"]);
    if (url.startsWith("/v1/tasks"))
      return response([
        { id: "a", state: "ready", definition: { goal: "ACTIVE" } },
        { id: "b", state: "completed", definition: { goal: "FINISHED" } },
      ]);
    if (url.startsWith("/v1/events"))
      return response({ events: [], next: 0, more: false });
    throw Error(url);
  });
  await run("token='credential';project='p';refresh()");
  assert.match(element("tasks").textContent, /ACTIVE/);
  assert.doesNotMatch(element("tasks").textContent, /FINISHED/);
  await run("$('filter-all').onclick()");
  assert.match(element("tasks").textContent, /FINISHED/);
});

test("refresh never fills an intentionally blank project in an open draft", async () => {
  const { run, element } = app(async (url) => {
    if (url === "/v1/projects") return response(["existing"]);
    if (url.startsWith("/v1/tasks")) return response([]);
    if (url.startsWith("/v1/events"))
      return response({ events: [], next: 0, more: false });
    throw Error(url);
  });
  await run("token='credential';project='existing';openCreate(true);refresh()");
  assert.equal(element("create-form").elements.project.value, "");
});

test("creation errors are visible inside the open dialog", async () => {
  const { run, element } = app(async () => {
    throw Error("offline");
  });
  run("openCreate()");
  element("create-form").fields = {
    goal: "work",
    project: "",
    constraints: "",
  };
  await run(
    "$('create-form').onsubmit({preventDefault(){},currentTarget:$('create-form')})",
  );
  assert.match(element("create-notice").textContent, /Could not reach/);
});

test("unchanged checkpoint refresh preserves the expanded detail DOM", async () => {
  const { run, element } = app(async (url) => {
    if (url.startsWith("/v1/context"))
      return response({
        task: {
          id: "a",
          state: "running",
          definition: { goal: "Work" },
          revision: 1,
        },
        dependencies: [],
      });
    throw Error(url);
  });
  await run("token='credential';project='p';selected='a';contextView()");
  const original = element("context").children[0];
  await run("contextView()");
  assert.equal(element("context").children[0], original);
});

test("finishing an old create does not clear a newer draft", async () => {
  let release;
  const gate = new Promise((resolve) => (release = resolve));
  const { run, element } = app(async (url) => {
    if (url === "/v1/apply") {
      await gate;
      return response({});
    }
    if (url === "/v1/projects") return response([]);
    throw Error(url);
  });
  run("token='credential';openCreate()");
  element("create-form").fields = {
    goal: "First",
    project: "",
    constraints: "",
  };
  const first = run(
    "$('create-form').onsubmit({preventDefault(){},currentTarget:$('create-form')})",
  );
  run("$('create-dialog').close();openCreate()");
  element("create-form").fields = {
    goal: "Second",
    project: "",
    constraints: "",
  };
  release();
  await first;
  assert.equal(element("create-dialog").open, true);
  assert.equal(element("create-form").fields.goal, "Second");
});

test("disconnect during bootstrap cannot reconnect or retain a late cookie", async () => {
  let release;
  const gate = new Promise((resolve) => (release = resolve));
  let logouts = 0;
  const { run, element } = app(
    async (url, opts) => {
      if (url === "/console/session" && opts.method === "POST") {
        await gate;
        return response({});
      }
      if (url === "/console/session" && opts.method === "DELETE") {
        logouts++;
        return response({});
      }
      if (url === "/console/info")
        return response({ endpoint: "http://127.0.0.1:7331" });
      if (url === "/v1/projects") return response([]);
      throw Error(url);
    },
    { hash: "#connect=once" },
  );
  run("$('disconnect').onclick()");
  release();
  await new Promise(setImmediate);
  assert.equal(element("workspace").hidden, true);
  assert.equal(run("localConnected"), false);
  assert.equal(logouts, 2);
});

test("automatic refresh preserves task button focus targets when nothing changed", async () => {
  const { run, element } = app(async (url) => {
    if (url === "/v1/projects") return response(["p"]);
    if (url.startsWith("/v1/tasks"))
      return response([
        { id: "a", state: "ready", definition: { goal: "Work" } },
      ]);
    if (url.startsWith("/v1/events"))
      return response({ events: [], next: 0, more: false });
    throw Error(url);
  });
  await run("token='credential';project='p';refresh()");
  const original = element("tasks").children[0];
  await run("refresh()");
  assert.equal(element("tasks").children[0], original);
});
