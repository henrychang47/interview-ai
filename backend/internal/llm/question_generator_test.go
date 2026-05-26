package llm

import (
	"context"
	"testing"
)

func TestMockQuestionGeneratorReturnsRequestedCount(t *testing.T) {
	generator := MockQuestionGenerator{}

	questions, err := generator.GenerateQuestions(context.Background(), GenerateQuestionsInput{
		JobTitle:       "後端工程師",
		JobDescription: "需要熟悉 Go、PostgreSQL、REST API",
		UserProfile:    "有 Java 和 Go 學習經驗",
		QuestionCount:  3,
	})

	if err != nil {
		t.Fatalf("GenerateQuestions returned error: %v", err)
	}
	if len(questions) != 3 {
		t.Fatalf("expected 3 questions, got %d", len(questions))
	}
	for index, question := range questions {
		expectedOrder := index + 1
		if question.Order != expectedOrder {
			t.Fatalf("expected order %d, got %d", expectedOrder, question.Order)
		}
		if question.Text == "" {
			t.Fatalf("expected question text at order %d", expectedOrder)
		}
	}
}
