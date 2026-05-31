import { useEffect, useState } from 'react'

import InterviewDetailPage from './pages/InterviewDetailPage'
import InterviewResultPage from './pages/InterviewResultPage'
import InterviewSessionPage from './pages/InterviewSessionPage'
import HomePage from './pages/HomePage'
import NewInterviewPage from './pages/NewInterviewPage'

function getRoute(pathname: string) {
  if (pathname === '/interviews/new') {
    return { name: 'new' as const }
  }

  const sessionMatch = pathname.match(/^\/interviews\/([^/]+)\/session$/)
  if (sessionMatch) {
    return { name: 'session' as const, interviewID: decodeURIComponent(sessionMatch[1]) }
  }

  const resultMatch = pathname.match(/^\/interviews\/([^/]+)\/result$/)
  if (resultMatch) {
    return { name: 'result' as const, interviewID: decodeURIComponent(resultMatch[1]) }
  }

  const detailMatch = pathname.match(/^\/interviews\/([^/]+)$/)
  if (detailMatch) {
    return { name: 'detail' as const, interviewID: decodeURIComponent(detailMatch[1]) }
  }

  return { name: 'home' as const }
}

export default function App() {
  const [route, setRoute] = useState(() => getRoute(window.location.pathname))

  useEffect(() => {
    function handlePopState() {
      setRoute(getRoute(window.location.pathname))
    }

    window.addEventListener('popstate', handlePopState)
    return () => window.removeEventListener('popstate', handlePopState)
  }, [])

  function navigate(path: string) {
    window.history.pushState({}, '', path)
    setRoute(getRoute(path))
  }

  if (route.name === 'new') {
    return <NewInterviewPage onCreated={(interviewID) => navigate(`/interviews/${interviewID}`)} />
  }

  if (route.name === 'session') {
    return <InterviewSessionPage interviewID={route.interviewID} />
  }

  if (route.name === 'result') {
    return <InterviewResultPage interviewID={route.interviewID} />
  }

  if (route.name === 'detail') {
    return <InterviewDetailPage interviewID={route.interviewID} />
  }

  return <HomePage />
}
