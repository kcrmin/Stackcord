# Discovery batches and progress

Read saved meaning before asking. Use `stackcord project discovery --draft <draft-root> --json`
before initialization or `stackcord project discovery --root <root> --json` afterward.
The command is read-only. Legacy checkpoints report unknown progress rather than inventing it.

`stackcord project checkpoint --help` shows the complete input shape. Optional `discovery`
contains `scope_ready`, ordered `sections` and `questions`. A section has a stable `id`,
`title`, `status` (`planned`, `active`, `complete`) and `estimated_remaining` upper estimate.
The lower bound is the number of known open questions. Re-estimate when scope changes.

Each open question has exactly one plan with `question_id`, `section_id`, `options`
(`id`, `label`), optional `recommended_option_id`, `requires_explicit_answer`, `blocking`,
and optional `depends_on` question IDs. Use no options for free-form-only questions.
Dependencies reference pending questions or accepted decisions' `question_id` values.
Completed sections have no pending questions and a zero remaining estimate.

Present independent routine questions together, keeping stable option IDs. In a supported
question UI, put recommended choices first and preselect only as proposals. Plain-text
fallback labels recommendations clearly. Never interpret silence, timeout or preselection
as submission. Security/permission changes, destructive actions, purchases and releases
require explicit answers and cannot be accepted through a routine-defaults shortcut.

After submitted answers, save only the accepted choices and reasons under `decisions`;
set each decision's `question_id`, remove the answered `open_questions` and its plan, and
update section estimates. Keep unanswered questions. Confirm the successful checkpoint,
then show accepted decisions separately from recommendations and remaining blockers.
If new evidence invalidates an answer, explain reopening, remove its old question link,
and retain the old rationale as a superseded decision record rather than silently erasing it.

Use the existing checkpoint workflow while in a draft. After initialization, canonical
question/decision Markdown lives in `specs/product/`; only section planning and question-ID
links live in `.harness/discovery.yaml`. Update those sources together in a reviewed change.
Do not copy a draft checkpoint over initialized product files or create competing state.

Show progress every round, including after host/session changes:
“현재 환경 설정 · 완료 1/3섹션 · 남은 섹션: 환경 설정, 정책 · 남은 질문 약 4–6개”.
Explain estimate changes. Allow nonblocking questions to remain for implementation.
`scope_ready` is the agent's recorded scope assessment; readiness also requires no pending
blocking question. Readiness does not grant policy approval, execute setup, or imply that
discovery is complete. Clearly say when the harness can be created and what remains.
