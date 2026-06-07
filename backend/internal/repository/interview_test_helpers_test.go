package repository

import (
	"context"

	"interview-ai/backend/internal/llm"
	"interview-ai/backend/internal/model"
)

func createInterviewWithQuestions(
	ctx context.Context,
	repository *InterviewRepository,
	input model.CreateInterviewRequest,
	questions []llm.GeneratedQuestion,
) (model.CreateInterviewResponse, error) {
	if input.QuestionLanguage == "" {
		input.QuestionLanguage = model.QuestionLanguageZhTW
	}

	created, err := repository.CreatePending(ctx, input)
	if err != nil {
		return model.CreateInterviewResponse{}, err
	}
	if err := repository.SaveGeneratedQuestions(ctx, created.ID, questions); err != nil {
		return model.CreateInterviewResponse{}, err
	}

	return model.CreateInterviewResponse{
		ID:     created.ID,
		Status: model.InterviewStatusQuestionsReady,
	}, nil
}
