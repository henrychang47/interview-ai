package llm

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"interview-ai/backend/internal/model"

	"google.golang.org/genai"
)

const (
	defaultGeminiTTSModel         = "gemini-3.1-flash-tts-preview"
	defaultGeminiTTSFallbackModel = "gemini-2.5-flash-preview-tts"
	defaultGeminiTTSVoice         = "Kore"
	geminiTTSSampleRate           = 24000
	geminiTTSBitsPerSample        = 16
	geminiTTSChannels             = 1
)

type QuestionTTSGenerator interface {
	GenerateQuestionSpeech(ctx context.Context, input model.QuestionTTSInput) ([]byte, error)
}

type GeminiQuestionTTSGeneratorConfig struct {
	APIKey        string
	Model         string
	FallbackModel string
	Voice         string
	Backoff       func(context.Context, int, *http.Response) error
	Logger        CallLogger
}

type GeminiQuestionTTSGenerator struct {
	apiKey        string
	model         string
	fallbackModel string
	voice         string
	models        geminiContentGenerator
	clientErr     error
	backoff       func(context.Context, int, *http.Response) error
	logger        CallLogger
}

func NewGeminiQuestionTTSGenerator(config GeminiQuestionTTSGeneratorConfig) *GeminiQuestionTTSGenerator {
	apiKey := strings.TrimSpace(config.APIKey)
	modelName := strings.TrimSpace(config.Model)
	if modelName == "" {
		modelName = defaultGeminiTTSModel
	}
	fallbackModel := strings.TrimSpace(config.FallbackModel)
	if fallbackModel == "" {
		fallbackModel = defaultGeminiTTSFallbackModel
	}
	voice := strings.TrimSpace(config.Voice)
	if voice == "" {
		voice = defaultGeminiTTSVoice
	}
	backoff := config.Backoff
	if backoff == nil {
		backoff = sleepBeforeGeminiRetry
	}

	var models geminiContentGenerator
	var clientErr error
	if apiKey != "" {
		models, clientErr = newGeminiModels(context.Background(), apiKey)
	}

	return &GeminiQuestionTTSGenerator{
		apiKey:        apiKey,
		model:         modelName,
		fallbackModel: fallbackModel,
		voice:         voice,
		models:        models,
		clientErr:     clientErr,
		backoff:       backoff,
		logger:        config.Logger,
	}
}

func (g *GeminiQuestionTTSGenerator) GenerateQuestionSpeech(ctx context.Context, input model.QuestionTTSInput) ([]byte, error) {
	if strings.TrimSpace(g.apiKey) == "" {
		return nil, ErrGeminiAPIKeyRequired
	}
	if g.clientErr != nil {
		return nil, fmt.Errorf("create Gemini client: %w", g.clientErr)
	}
	if g.models == nil {
		return nil, errors.New("Gemini client is not initialized")
	}

	models := []string{g.model}
	if g.fallbackModel != "" && g.fallbackModel != g.model {
		models = append(models, g.fallbackModel)
	}

	var lastErr error
	for _, modelName := range models {
		audio, err := g.generateQuestionSpeechWithModel(ctx, modelName, input)
		if err == nil {
			return audio, nil
		}
		lastErr = err
		if !isTransientGeminiError(err) {
			return nil, err
		}
	}

	return nil, lastErr
}

func (g *GeminiQuestionTTSGenerator) generateQuestionSpeechWithModel(ctx context.Context, modelName string, input model.QuestionTTSInput) ([]byte, error) {
	var lastErr error
	for attempt := 1; attempt <= defaultGeminiMaxAttempts; attempt++ {
		callResult, audio, err := g.callGeminiTTS(ctx, modelName, input)
		if err != nil {
			g.logQuestionTTSCall(ctx, modelName, input, failedGeminiLogStatus(err), callResult, err)
			lastErr = err
			if !isTransientGeminiError(err) || attempt == defaultGeminiMaxAttempts {
				return nil, err
			}
			if backoffErr := g.backoff(ctx, attempt, nil); backoffErr != nil {
				return nil, backoffErr
			}
			continue
		}
		g.logQuestionTTSCall(ctx, modelName, input, model.LLMCallStatusSuccess, callResult, nil)
		return audio, nil
	}

	return nil, lastErr
}

func (g *GeminiQuestionTTSGenerator) callGeminiTTS(ctx context.Context, modelName string, input model.QuestionTTSInput) (geminiCallResult, []byte, error) {
	start := time.Now()
	response, err := g.models.GenerateContent(
		ctx,
		strings.TrimPrefix(modelName, "models/"),
		genai.Text(buildQuestionTTSPrompt(input)),
		&genai.GenerateContentConfig{
			ResponseModalities: []string{"AUDIO"},
			SpeechConfig: &genai.SpeechConfig{
				VoiceConfig: &genai.VoiceConfig{
					PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{
						VoiceName: g.voice,
					},
				},
			},
		},
	)
	callResult := geminiCallResult{latencyMS: elapsedMilliseconds(start)}
	if err != nil {
		return callResult, nil, err
	}
	callResult.inputTokens, callResult.outputTokens, callResult.totalTokens = geminiUsageTokenCounts(response)

	audioBytes, err := firstInlineAudioBytes(response)
	if err != nil {
		return callResult, nil, err
	}
	return callResult, wrapPCMAsWAV(audioBytes), nil
}

func buildQuestionTTSPrompt(input model.QuestionTTSInput) string {
	languageInstruction := "繁體中文"
	if input.QuestionLanguage == model.QuestionLanguageEnUS {
		languageInstruction = "English"
	}
	return fmt.Sprintf("請朗讀以下面試題目。請使用自然、清楚、沉穩的%s口吻，只朗讀題目文字，不要加入說明、答案或額外內容。\n\n%s", languageInstruction, input.QuestionText)
}

func firstInlineAudioBytes(response *genai.GenerateContentResponse) ([]byte, error) {
	if response == nil {
		return nil, fmt.Errorf("%w: missing TTS response", ErrInvalidLLMResponse)
	}
	for _, candidate := range response.Candidates {
		if candidate == nil || candidate.Content == nil {
			continue
		}
		for _, part := range candidate.Content.Parts {
			if part != nil && part.InlineData != nil && len(part.InlineData.Data) > 0 {
				return part.InlineData.Data, nil
			}
		}
	}
	return nil, fmt.Errorf("%w: missing TTS audio", ErrInvalidLLMResponse)
}

func wrapPCMAsWAV(pcm []byte) []byte {
	byteRate := uint32(geminiTTSSampleRate * geminiTTSChannels * geminiTTSBitsPerSample / 8)
	blockAlign := uint16(geminiTTSChannels * geminiTTSBitsPerSample / 8)
	dataSize := uint32(len(pcm))

	var buffer bytes.Buffer
	buffer.Grow(44 + len(pcm))
	buffer.WriteString("RIFF")
	_ = binary.Write(&buffer, binary.LittleEndian, uint32(36)+dataSize)
	buffer.WriteString("WAVE")
	buffer.WriteString("fmt ")
	_ = binary.Write(&buffer, binary.LittleEndian, uint32(16))
	_ = binary.Write(&buffer, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buffer, binary.LittleEndian, uint16(geminiTTSChannels))
	_ = binary.Write(&buffer, binary.LittleEndian, uint32(geminiTTSSampleRate))
	_ = binary.Write(&buffer, binary.LittleEndian, byteRate)
	_ = binary.Write(&buffer, binary.LittleEndian, blockAlign)
	_ = binary.Write(&buffer, binary.LittleEndian, uint16(geminiTTSBitsPerSample))
	buffer.WriteString("data")
	_ = binary.Write(&buffer, binary.LittleEndian, dataSize)
	buffer.Write(pcm)
	return buffer.Bytes()
}

func (g *GeminiQuestionTTSGenerator) logQuestionTTSCall(ctx context.Context, modelName string, input model.QuestionTTSInput, status string, result geminiCallResult, err error) {
	log := model.LLMCallLog{
		Operation:    model.LLMOperationGenerateQuestionTTS,
		Provider:     model.LLMProviderGemini,
		Model:        strings.TrimPrefix(modelName, "models/"),
		Status:       status,
		LatencyMS:    intPtr(result.latencyMS),
		InputTokens:  result.inputTokens,
		OutputTokens: result.outputTokens,
		TotalTokens:  result.totalTokens,
		ErrorCode:    geminiErrorCode(err),
		ErrorMessage: geminiErrorMessage(err),
	}
	if strings.TrimSpace(input.InterviewID) != "" {
		log.InterviewID = stringPtr(input.InterviewID)
	}
	if strings.TrimSpace(input.QuestionID) != "" {
		log.QuestionID = stringPtr(input.QuestionID)
	}
	logGeminiCall(ctx, g.logger, log)
}
