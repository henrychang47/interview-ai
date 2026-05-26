package llm

import (
	"context"
	"fmt"
)

type QuestionGenerator interface {
	GenerateQuestions(ctx context.Context, input GenerateQuestionsInput) ([]GeneratedQuestion, error)
}

type GenerateQuestionsInput struct {
	JobTitle       string
	JobDescription string
	UserProfile    string
	QuestionCount  int
}

type GeneratedQuestion struct {
	Order int
	Text  string
}

type MockQuestionGenerator struct{}

func (MockQuestionGenerator) GenerateQuestions(ctx context.Context, input GenerateQuestionsInput) ([]GeneratedQuestion, error) {
	baseQuestions := []string{
		"請介紹你過去與後端開發相關的經驗。",
		"你如何設計一個 REST API？",
		"你使用 PostgreSQL 時會注意哪些事情？",
		"請分享你排查後端服務問題的經驗。",
		"你如何確保 API 的可維護性與可測試性？",
		"請說明你如何處理資料庫交易與錯誤回復。",
		"你如何設計一個可擴充的服務架構？",
		"請分享你與跨職能團隊合作的經驗。",
		"你如何評估系統效能瓶頸？",
		"請描述你學習新技術並應用到專案中的方式。",
	}

	questions := make([]GeneratedQuestion, 0, input.QuestionCount)
	for index := 0; index < input.QuestionCount; index++ {
		text := baseQuestions[index%len(baseQuestions)]
		if input.JobTitle != "" && index >= len(baseQuestions) {
			text = fmt.Sprintf("針對%s，%s", input.JobTitle, text)
		}
		questions = append(questions, GeneratedQuestion{
			Order: index + 1,
			Text:  text,
		})
	}

	return questions, nil
}
