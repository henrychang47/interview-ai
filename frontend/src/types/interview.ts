export type CreateInterviewRequest = {
  job_title: string
  job_description: string
  user_profile: string
  question_count: number
}

export type CreateInterviewResponse = {
  id: string
  status: string
}

export type Question = {
  id: string
  order: number
  text: string
}

export type Answer = {
  id: string
  question_id: string
  audio_path: string | null
  transcript_text: string | null
  created_at: string
}

export type UploadAnswerResponse = {
  id: string
  interview_id: string
  question_id: string
  audio_path: string
  transcript_text: string | null
}

export type InterviewDetail = {
  id: string
  job_title: string
  job_description: string
  user_profile: string
  question_count: number
  status: string
  questions: Question[]
  answers: Answer[]
}
