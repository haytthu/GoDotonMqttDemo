export async function api(path, { method = 'GET', body } = {}) {
  const options = { method }
  if (body instanceof FormData) {
    options.body = body
  } else if (body !== undefined) {
    options.headers = { 'Content-Type': 'application/json' }
    options.body = JSON.stringify(body)
  }
  let response
  try {
    response = await fetch('/api' + path, options)
  } catch (cause) {
    throw new Error('Unable to reach the backend', { cause })
  }
  let text
  try {
    text = await response.text()
  } catch (cause) {
    throw new Error('Unable to read the backend response', { cause })
  }
  let payload = null
  if (text) {
    try {
      payload = JSON.parse(text)
    } catch {
      payload = null
    }
  }
  if (!response.ok) {
    throw new Error(payload?.message || payload?.error || `Request failed (HTTP ${response.status})`)
  }
  if (text && payload === null && text !== 'null') {
    throw new Error('The backend returned an unreadable response')
  }
  return payload
}

export function query(params) {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== '' && value != null) search.set(key, value)
  }
  return search.size ? '?' + search.toString() : ''
}

