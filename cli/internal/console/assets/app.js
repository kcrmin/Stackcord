"use strict";
const $ = (id) => document.getElementById(id);
const en = {
  tagline: "Work continues. Sessions change.",
  connect: "Connect to your runtime",
  credentialHelp:
    "Use the credential configured when starting the runtime. It stays in this page’s memory.",
  credential: "Runtime credential",
  connectButton: "Connect",
  overview: "COORDINATION",
  title: "A shared place for ongoing work.",
  disconnect: "Disconnect",
  project: "Project",
  budget: "Context byte budget",
  auto: "Refresh every 5 seconds",
  refresh: "Refresh",
  tasks: "Tasks",
  taskHelp:
    "Ownership is per session. Expired leases may be reclaimed; they do not prove a process stopped.",
  handoff: "Current handoff",
  handoffHelp:
    "Select a task to see its checkpoint, constraints and referenced results.",
  activity: "Coordination activity",
  activityHelp:
    "Recorded requests and state changes. No private model reasoning or generated chat.",
  more: "Load more changes",
  create: "Create a task",
  projectId: "Project ID",
  taskId: "Task ID",
  goal: "Goal",
  constraints: "Constraints (one per line)",
  boundary:
    "Reported token counts can be incomplete. The runtime does not execute models, reset quotas or verify Git changes. Back up its database separately from Git.",
  empty: "No tasks yet. Create the first one below.",
  requeue: "Requeue this task",
  retryHelp:
    "Requeue allows a worker to claim the task again. Check partial changes and any required approvals first.",
  unknown: "No reported token usage",
  tokens: "Reported tokens",
  created: "Task created",
  requeued: "Task requeued",
  expired: "lease expired",
  unowned: "unassigned",
  summary: "Completed",
  next: "Next step",
  blockers: "Blockers",
  references: "References",
  dependencies: "Dependencies",
  apiDetails: "API details",
  owner: "Owner",
  taskConstraints: "Constraints",
  ready: "Ready",
  running: "Running",
  paused: "Paused",
  completed: "Completed",
  failed: "Failed",
};
const ko = {
  tagline: "세션이 바뀌어도 작업은 이어집니다.",
  connect: "실행 서비스에 연결",
  credentialHelp:
    "실행 서비스를 시작할 때 설정한 인증값을 입력하세요. 이 페이지의 메모리에만 보관합니다.",
  credential: "서비스 인증값",
  connectButton: "연결",
  overview: "작업 조율",
  title: "진행 중인 작업을 함께 보는 공간.",
  disconnect: "연결 해제",
  project: "프로젝트",
  budget: "복귀 문맥 바이트 예산",
  auto: "5초마다 새로고침",
  refresh: "새로고침",
  tasks: "작업",
  taskHelp:
    "담당 권한은 세션마다 관리합니다. 만료된 권한을 인수할 수 있지만 이전 프로세스의 종료를 보장하지는 않습니다.",
  handoff: "현재 인수인계",
  handoffHelp:
    "작업을 선택하면 중단 지점, 제약 조건, 결과물 참조를 볼 수 있습니다.",
  activity: "작업 통신 기록",
  activityHelp:
    "실제로 기록된 요청과 상태 변화입니다. 모델 내부 사고나 생성한 채팅이 아닙니다.",
  more: "변경 더 읽기",
  create: "작업 만들기",
  projectId: "프로젝트 ID",
  taskId: "작업 ID",
  goal: "목표",
  constraints: "제약 조건 (한 줄에 하나)",
  boundary:
    "보고된 토큰 수는 일부 사용량만 포함할 수 있습니다. 실행 서비스는 모델 실행·한도 초기화·Git 변경 검증을 하지 않습니다. 데이터베이스는 Git과 별도로 백업하세요.",
  empty: "아직 작업이 없습니다. 아래에서 첫 작업을 만드세요.",
  requeue: "작업 다시 배정하기",
  retryHelp:
    "다른 실행기가 작업을 맡을 수 있게 합니다. 부분 변경과 필요한 승인을 먼저 확인하세요.",
  unknown: "보고된 사용량 없음",
  tokens: "보고된 토큰",
  created: "작업을 만들었습니다",
  requeued: "작업을 다시 배정할 수 있습니다",
  expired: "담당 권한 만료",
  unowned: "미배정",
  summary: "완료한 내용",
  next: "다음 단계",
  blockers: "차단 사유",
  references: "결과물과 입력",
  dependencies: "선행 작업",
  apiDetails: "API 상세",
  owner: "담당자",
  taskConstraints: "제약 조건",
  ready: "준비",
  running: "진행 중",
  paused: "중단",
  completed: "완료",
  failed: "실패",
};
let language = navigator.language.startsWith("ko") ? "ko" : "en",
  token = "",
  project = "",
  cursor = 0,
  selected = "",
  busy = false,
  pendingRefresh = false,
  generation = 0;
const t = (k) => (language === "ko" ? ko : en)[k];
function translate() {
  document.documentElement.lang = language;
  document
    .querySelectorAll("[data-i18n]")
    .forEach((e) => (e.textContent = t(e.dataset.i18n)));
  $("language").textContent = language === "ko" ? "English" : "한국어";
}
function notice(e) {
  $("notice").textContent = e instanceof Error ? e.message : String(e || "");
}
function node(tag, text, cls) {
  const e = document.createElement(tag);
  e.textContent = text;
  if (cls) e.className = cls;
  return e;
}
async function api(path, input) {
  const response = await fetch("/v1/" + path, {
    method: input ? "POST" : "GET",
    credentials: "omit",
    cache: "no-store",
    headers: {
      Authorization: "Bearer " + token,
      ...(input ? { "Content-Type": "application/json" } : {}),
    },
    ...(input ? { body: JSON.stringify(input) } : {}),
  });
  const result = await response.json();
  if (!response.ok) throw Error(result.error || response.statusText);
  return result;
}
async function contextView() {
  if (!project || !selected) return;
  const gen = generation,
    p = project,
    id = selected;
  const value = await api(
    "context?" +
      new URLSearchParams({
        project: p,
        task: id,
        max_bytes: $("budget").value,
      }),
  );
  if (gen !== generation || p !== project || id !== selected) return;
  const view = $("context");
  view.replaceChildren();
  const task = value.task;
  view.append(node("h3", task.definition.goal));
  view.append(
    node(
      "p",
      `${t(task.state)} · ${t("owner")}: ${task.owner || t("unowned")}`,
      "muted",
    ),
  );
  view.append(
    node(
      "p",
      task.reported_usage
        ? `${t("tokens")}: ${task.reported_usage.input_tokens} / ${task.reported_usage.output_tokens}`
        : t("unknown"),
      "muted",
    ),
  );
  const section = (label, lines) => {
    if (!lines?.length) return;
    view.append(node("h4", t(label)));
    const list = node("ul", "");
    for (const line of lines) list.append(node("li", line));
    view.append(list);
  };
  section("summary", task.checkpoint?.summary ? [task.checkpoint.summary] : []);
  section("next", task.checkpoint?.next ? [task.checkpoint.next] : []);
  section("taskConstraints", task.definition.constraints);
  section("blockers", task.checkpoint?.blockers);
  section(
    "references",
    [
      ...(task.definition.inputs || []),
      ...(task.checkpoint?.artifacts || []),
    ].map((ref) => ref.uri),
  );
  section(
    "dependencies",
    value.dependencies.map((dep) => `${dep.id} · ${t(dep.state)}`),
  );
  const details = node("details", "");
  details.append(
    node("summary", t("apiDetails")),
    node("pre", JSON.stringify(value, null, 2)),
  );
  view.append(details);
  if (["paused", "failed"].includes(task.state)) {
    view.append(node("p", t("retryHelp"), "muted"));
    const button = node("button", t("requeue"));
    button.type = "button";
    button.onclick = async () => {
      button.disabled = true;
      try {
        await api("apply", {
          id: crypto.randomUUID(),
          project: p,
          task: id,
          kind: "requeue",
          expected_revision: task.revision,
        });
        notice(t("requeued"));
        await refresh();
      } catch (e) {
        notice(e);
      } finally {
        button.disabled = false;
      }
    };
    view.append(button);
  }
}
async function refresh() {
  if (!token) return;
  if (busy) {
    pendingRefresh = true;
    return;
  }
  busy = true;
  const gen = generation;
  try {
    const ids = await api("projects");
    if (gen !== generation) return;
    const selector = $("project");
    selector.replaceChildren(
      ...ids.map((id) => {
        const option = node("option", id);
        option.value = id;
        return option;
      }),
    );
    if (!ids.includes(project)) project = ids[0] || "";
    selector.value = project;
    if (!$("create-form").elements.project.value)
      $("create-form").elements.project.value = project;
    if (!project) {
      $("tasks").replaceChildren(node("p", t("empty"), "muted"));
      return;
    }
    const p = project;
    const tasks = await api("tasks?" + new URLSearchParams({ project: p }));
    if (gen !== generation || p !== project) return;
    $("tasks").replaceChildren();
    for (const task of tasks) {
      const button = node("button", "", "task");
      button.type = "button";
      button.append(
        node("strong", task.definition.goal),
        node("span", t(task.state), "state"),
        node(
          "span",
          `${task.id} · ${task.owner || t("unowned")}${task.state === "running" && Date.parse(task.lease_until) <= Date.now() ? " · " + t("expired") : ""}`,
        ),
      );
      button.onclick = () => {
        selected = task.id;
        $("context").replaceChildren();
        contextView().catch(notice);
      };
      $("tasks").append(button);
    }
    if (!tasks.length) $("tasks").append(node("p", t("empty"), "muted"));
    const page = await api(
      "events?" +
        new URLSearchParams({
          project: p,
          after: String(cursor),
          limit: "100",
        }),
    );
    if (gen !== generation || p !== project) return;
    for (const event of page.events) {
      $("events").append(
        node(
          "div",
          `${event.sequence} · ${event.task} · ${event.kind} → ${event.state}${event.owner ? " · " + event.owner : ""}`,
          "event",
        ),
      );
      while ($("events").children.length > 300) $("events").firstChild.remove();
    }
    cursor = page.next;
    $("more").hidden = !page.more;
    await contextView();
  } catch (e) {
    if (gen === generation) notice(e);
  } finally {
    busy = false;
    if (pendingRefresh) {
      pendingRefresh = false;
      queueMicrotask(refresh);
    }
  }
}
$("login-form").onsubmit = async (e) => {
  e.preventDefault();
  token = $("token").value;
  $("token").value = "";
  try {
    await api("projects");
    $("login").hidden = true;
    $("workspace").hidden = false;
    notice("");
    await refresh();
  } catch (e) {
    token = "";
    notice(e);
  }
};
$("disconnect").onclick = () => {
  generation++;
  token = "";
  project = "";
  selected = "";
  cursor = 0;
  $("workspace").hidden = true;
  $("login").hidden = false;
  for (const id of ["tasks", "context", "events", "project"])
    $(id).replaceChildren();
  $("create-form").reset();
  notice("");
};
$("project").onchange = () => {
  generation++;
  project = $("project").value;
  $("create-form").elements.project.value = project;
  cursor = 0;
  selected = "";
  $("tasks").replaceChildren();
  $("events").replaceChildren();
  $("context").replaceChildren();
  refresh();
};
$("create-form").onsubmit = async (e) => {
  e.preventDefault();
  const form = e.currentTarget,
    button = form.querySelector("button");
  button.disabled = true;
  const fields = new FormData(form);
  try {
    await api("apply", {
      id: crypto.randomUUID(),
      project: fields.get("project").trim(),
      task: fields.get("task").trim(),
      kind: "create",
      definition: {
        goal: fields.get("goal"),
        constraints: fields
          .get("constraints")
          .split("\n")
          .map((s) => s.trim())
          .filter(Boolean),
      },
    });
    const next = fields.get("project").trim();
    if (project !== next) {
      generation++;
      cursor = 0;
      $("events").replaceChildren();
    }
    project = next;
    selected = fields.get("task").trim();
    form.reset();
    notice(t("created"));
    await refresh();
  } catch (e) {
    notice(e);
  } finally {
    button.disabled = false;
  }
};
$("refresh").onclick = () => refresh();
$("more").onclick = () => refresh();
$("budget").onchange = () => contextView().catch(notice);
$("language").onclick = () => {
  language = language === "ko" ? "en" : "ko";
  translate();
  refresh();
};
setInterval(() => {
  if ($("auto").checked && !document.hidden) refresh();
}, 5000);
translate();
