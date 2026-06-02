const questionAudioCache = new Map<string, Blob>()
const prefetchedInterviewIDs = new Set<string>()

function cacheKey(interviewID: string, questionID: string) {
  return `${interviewID}:${questionID}`
}

export function storeQuestionAudio(interviewID: string, questionID: string, audio: Blob) {
  questionAudioCache.set(cacheKey(interviewID, questionID), audio)
}

export function getQuestionAudio(interviewID: string, questionID: string) {
  return questionAudioCache.get(cacheKey(interviewID, questionID)) ?? null
}

export function markQuestionAudioPrefetchComplete(interviewID: string) {
  prefetchedInterviewIDs.add(interviewID)
}

export function hasQuestionAudioPrefetchCompleted(interviewID: string) {
  return prefetchedInterviewIDs.has(interviewID)
}

export function clearQuestionAudioCache() {
  questionAudioCache.clear()
  prefetchedInterviewIDs.clear()
}
