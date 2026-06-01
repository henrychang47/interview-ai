package llm

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"interview-ai/backend/internal/model"

	"google.golang.org/genai"
)

type CallLogger interface {
	CreateLLMCallLog(ctx context.Context, log model.LLMCallLog) error
}

type geminiCallResult struct {
	text         string
	latencyMS    int
	inputTokens  *int
	outputTokens *int
	totalTokens  *int
}

func geminiUsageTokenCounts(response *genai.GenerateContentResponse) (*int, *int, *int) {
	if response == nil || response.UsageMetadata == nil {
		return nil, nil, nil
	}

	return int32PtrToIntPtr(response.UsageMetadata.PromptTokenCount),
		int32PtrToIntPtr(response.UsageMetadata.CandidatesTokenCount),
		int32PtrToIntPtr(response.UsageMetadata.TotalTokenCount)
}

func int32PtrToIntPtr(value int32) *int {
	if value == 0 {
		return nil
	}
	result := int(value)
	return &result
}

func stringPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func intPtr(value int) *int {
	return &value
}

func logGeminiCall(ctx context.Context, logger CallLogger, log model.LLMCallLog) {
	if logger == nil {
		return
	}
	if log.Provider == "" {
		log.Provider = model.LLMProviderGemini
	}
	if log.LatencyMS == nil {
		log.LatencyMS = intPtr(0)
	}
	_ = logger.CreateLLMCallLog(context.WithoutCancel(ctx), log)
}

func failedGeminiLogStatus(err error) string {
	if err == nil {
		return model.LLMCallStatusSuccess
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return model.LLMCallStatusTimeout
	}
	var apiErr genai.APIError
	if errors.As(err, &apiErr) && apiErr.Code == http.StatusTooManyRequests {
		return model.LLMCallStatusRateLimited
	}
	return model.LLMCallStatusFailed
}

func geminiErrorCode(err error) *string {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return stringPtr("context_deadline_exceeded")
	}
	var apiErr genai.APIError
	if errors.As(err, &apiErr) && apiErr.Code != 0 {
		return stringPtr(strconv.Itoa(apiErr.Code))
	}
	return stringPtr(strings.ReplaceAll(strings.ToLower(err.Error()), " ", "_"))
}

func geminiErrorMessage(err error) *string {
	if err == nil {
		return nil
	}
	message := strings.TrimSpace(err.Error())
	if len(message) > 500 {
		message = message[:500]
	}
	return stringPtr(message)
}

func elapsedMilliseconds(start time.Time) int {
	return int(time.Since(start).Milliseconds())
}
