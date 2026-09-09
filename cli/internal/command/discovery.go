package command

import (
	"fmt"
	"strings"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/project"
	"github.com/kcrmin/Stackcord/cli/internal/workspace"
	"github.com/spf13/cobra"
)

func newDiscoveryCommand(version string, jsonOutput *bool) *cobra.Command {
	var draft, root, locale string
	cmd := &cobra.Command{Use: "discovery", Short: "Show saved question batches, decisions, and discovery progress", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if (draft == "") == (root == "") {
			return fmt.Errorf("provide exactly one of --draft or --root")
		}
		if locale != "" && locale != "en" && locale != "ko" {
			return fmt.Errorf("locale must be en or ko")
		}
		path := draft
		if root != "" {
			located, err := workspace.FindRoot(cmd.Context(), root)
			if err != nil {
				return err
			}
			path = located.Path
		}
		checkpoint, storedLocale, err := project.ReadDiscovery(path, draft != "")
		if err != nil {
			return err
		}
		if locale == "" {
			locale = storedLocale
		}
		report, err := project.SummarizeDiscovery(checkpoint)
		if err != nil {
			return err
		}
		return writeResult(cmd, *jsonOutput, discoveryResult(version, checkpoint, report, locale))
	}}
	cmd.Flags().StringVar(&draft, "draft", "", "saved discovery draft directory")
	cmd.Flags().StringVar(&root, "root", "", "initialized project or a path inside it")
	cmd.Flags().StringVar(&locale, "locale", "", "display language en|ko; defaults to saved locale")
	return cmd
}

func discoveryResult(version string, c project.DiscoveryCheckpoint, r project.DiscoveryReport, locale string) domain.Result {
	result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "project.discovery", OperationID: "discovery-read-only", Status: domain.StatusPassed, ExitCode: domain.ExitSuccess}
	ko := locale == "ko"
	lines := []string{}
	if !r.ProgressKnown {
		result.Facts = append(result.Facts, domain.Item{Code: "discovery.progress", Message: "unknown"})
		if ko {
			lines = append(lines, "질문 진행 정보 없음: 저장된 결정을 유지하고 남은 질문을 정리하세요.")
		} else {
			lines = append(lines, "Discovery progress unknown: preserve saved decisions and organize remaining questions.")
		}
	} else {
		titles := []string{}
		for _, s := range r.CompletedSections {
			result.Facts = append(result.Facts, domain.Item{Code: "discovery.section.complete", Message: s.Title, Refs: []string{s.ID}})
		}
		for _, s := range r.RemainingSections {
			titles = append(titles, s.Title)
			result.Facts = append(result.Facts, domain.Item{Code: "discovery.section.remaining", Message: s.Title, Refs: []string{s.ID}})
		}
		current := "complete"
		if r.CurrentSection != nil {
			current = r.CurrentSection.Title
			result.Facts = append(result.Facts, domain.Item{Code: "discovery.section.current", Message: current, Refs: []string{r.CurrentSection.ID}})
		}
		estimate := fmt.Sprintf("%d–%d", r.RemainingMin, r.RemainingMax)
		result.Facts = append(result.Facts, domain.Item{Code: "discovery.questions.remaining", Message: estimate})
		if ko {
			lines = append(lines, fmt.Sprintf("현재 %s · 완료 %d/%d섹션 · 남은 질문 약 %s개", current, len(r.CompletedSections), len(c.Discovery.Sections), estimate), "남은 섹션: "+strings.Join(titles, ", "))
		} else {
			lines = append(lines, fmt.Sprintf("Current: %s · %d/%d sections complete · approximately %s questions remaining", current, len(r.CompletedSections), len(c.Discovery.Sections), estimate), "Remaining sections: "+strings.Join(titles, ", "))
		}
	}
	ready := "not_ready"
	if r.HarnessReady {
		ready = "ready"
	}
	result.Facts = append(result.Facts, domain.Item{Code: "discovery.harness-readiness", Message: ready, Refs: r.BlockingQuestions})
	if ko {
		lines = append(lines, "하네스 생성 준비: "+ready+" (승인·실행 권한과 별개)", "결정사항:")
	} else {
		lines = append(lines, "Harness readiness: "+ready+" (not approval or authorization)", "Accepted decisions:")
	}
	for _, d := range r.Decisions {
		result.Facts = append(result.Facts, domain.Item{Code: "discovery.decision.accepted", Message: d.Choice, Refs: []string{d.ID}})
		lines = append(lines, "- "+d.Choice+" — "+d.Rationale)
	}
	prompts := map[string]string{}
	for _, q := range c.OpenQuestions {
		prompts[q.ID] = q.Summary
	}
	addQuestion := func(q project.DiscoveryQuestion, kind string) {
		result.NextActions = append(result.NextActions, domain.Item{Code: "discovery.question." + kind, Message: prompts[q.QuestionID], Refs: []string{q.QuestionID, q.SectionID}})
		lines = append(lines, "- "+prompts[q.QuestionID])
		// Show the recommendation first, while keeping the original stable option IDs.
		ordered := append([]project.DiscoveryOption{}, q.Options...)
		for i, o := range ordered {
			if o.ID == q.RecommendedOptionID {
				ordered = append([]project.DiscoveryOption{o}, append(ordered[:i], ordered[i+1:]...)...)
				break
			}
		}
		for _, o := range ordered {
			code, label := "discovery.option", o.Label
			if o.ID == q.RecommendedOptionID {
				code = "discovery.recommendation"
				if ko {
					label += " (권장·미확정)"
				} else {
					label += " (recommended, not accepted)"
				}
			}
			result.Facts = append(result.Facts, domain.Item{Code: code, Message: o.Label, Refs: []string{q.QuestionID, o.ID}})
			lines = append(lines, "  "+o.ID+": "+label)
		}
	}
	if len(r.Batch) > 0 {
		if ko {
			lines = append(lines, "함께 답할 질문 (권장 답변은 제출 전까지 미확정):")
		} else {
			lines = append(lines, "Answer together (recommended choices remain unaccepted until submitted):")
		}
	}
	for _, q := range r.Batch {
		addQuestion(q, "batch")
	}
	if len(r.ExplicitQuestions) > 0 {
		if ko {
			lines = append(lines, "명시적 답변이 필요한 질문:")
		} else {
			lines = append(lines, "Questions requiring an explicit answer:")
		}
	}
	for _, q := range r.ExplicitQuestions {
		addQuestion(q, "explicit")
	}
	if len(r.Prerequisites) > 0 {
		if ko {
			lines = append(lines, "다른 섹션에서 먼저 답해야 할 질문:")
		} else {
			lines = append(lines, "Answer these prerequisites in another section first:")
		}
		for _, q := range r.Prerequisites {
			addQuestion(q, "prerequisite")
		}
	}
	if r.ProgressKnown && len(r.Batch)+len(r.ExplicitQuestions)+len(r.Prerequisites) == 0 && len(r.RemainingSections) > 0 {
		message := "Review the current section and refine its remaining question estimate."
		if ko {
			message = "현재 섹션을 검토하고 남은 질문과 예상 수를 갱신하세요."
		}
		lines = append(lines, message)
		result.NextActions = append(result.NextActions, domain.Item{Code: "discovery.section.refine", Message: message, Refs: []string{r.CurrentSection.ID}})
	}
	if !r.ProgressKnown {
		for _, q := range c.OpenQuestions {
			addQuestion(project.DiscoveryQuestion{QuestionID: q.ID}, "explicit")
		}
	}
	if ko {
		lines = append(lines, "직접 입력도 가능합니다. 답변 후 결정사항과 진행 정보를 저장하세요.")
	} else {
		lines = append(lines, "Free-form answers are welcome. Save decisions and progress after answering.")
	}
	result.Summary = strings.Join(lines, "\n")
	return result
}
