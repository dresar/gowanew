import { http, results } from '@/lib/http'

export interface NewsletterText {
  text: string
  id: string
}

export interface NewsletterThreadMetadata {
  creation_time: string
  name: NewsletterText
  description: NewsletterText
}

export interface NewsletterViewerMetadata {
  role: string
  mute: string
}

export interface NewsletterMetadata {
  id: string
  thread_metadata: NewsletterThreadMetadata
  viewer_metadata: NewsletterViewerMetadata | null
}

export interface NewsletterMessage {
  server_id: number
  message_id: string
  type: string
  timestamp: string
  views_count: number
  reaction_counts?: Record<string, number>
  text?: string
}

export interface NewsletterMessagesResponse {
  data: NewsletterMessage[]
}

export interface NewsletterMessagesParams {
  newsletter_id: string
  count?: number
  before?: number
}

export async function unfollowNewsletter(newsletter_id: string): Promise<void> {
  await http.post('/newsletter/unfollow', { newsletter_id })
}

export function getNewsletterMessages(
  params: NewsletterMessagesParams,
): Promise<NewsletterMessagesResponse> {
  return results<NewsletterMessagesResponse>(http.get('/newsletter/messages', { params }))
}
