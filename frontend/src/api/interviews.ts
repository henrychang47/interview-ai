import type {
  CreateInterviewRequest,
  CreateInterviewResponse,
  InterviewDetail,
  UploadAnswerResponse,
} from '../types/interview'

const API_BASE_URL = ''

async function parseJSONResponse<T>(response: Response, fallbackMessage: string): Promise<T> {
  let body: unknown = null

  try {
    body = await response.json()
  } catch {
    body = null
  }

  if (!response.ok) {
    const message =
      body && typeof body === 'object' && 'error' in body && typeof body.error === 'string'
        ? body.error
        : fallbackMessage
    throw new Error(message)
  }

  return body as T
}

export async function createInterview(
  input: CreateInterviewRequest,
): Promise<CreateInterviewResponse> {
  const response = await fetch(`${API_BASE_URL}/api/interviews`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(input),
  })

  return parseJSONResponse<CreateInterviewResponse>(response, '建立面試失敗')
}

export async function getInterview(interviewID: string): Promise<InterviewDetail> {
  const response = await fetch(`${API_BASE_URL}/api/interviews/${interviewID}`)

  return parseJSONResponse<InterviewDetail>(response, '載入面試失敗')
}

export async function uploadAnswerAudio(
  interviewID: string,
  questionID: string,
  audio: Blob,
): Promise<UploadAnswerResponse> {
  const formData = new FormData()
  formData.append('audio', audio, 'answer.webm')

  const response = await fetch(
    `${API_BASE_URL}/api/interviews/${interviewID}/questions/${questionID}/answer`,
    {
      method: 'POST',
      body: formData,
    },
  )

  return parseJSONResponse<UploadAnswerResponse>(response, '上傳回答失敗')
}
