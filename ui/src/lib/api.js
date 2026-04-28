const BASE = '/api/v1'

function toRFC3339(localDatetime) {
  if (!localDatetime) return ''
  // datetime-local gives "2026-04-02T13:01:10", we need RFC3339 with timezone
  // toISOString() adds milliseconds (.000Z) which time.RFC3339 in Go rejects
  const d = new Date(localDatetime)
  return d.toISOString().replace(/\.\d+Z$/, 'Z')
}

async function get(path, params = {}) {
  const url = new URL(BASE + path, location.origin)
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== '') url.searchParams.set(k, v)
  }
  const res = await fetch(url)
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export const api = {
  services: () => get('/services'),

  logs: (params) => {
    const p = { ...params }
    if (p.from) p.from = toRFC3339(p.from)
    if (p.to) p.to = toRFC3339(p.to)
    return get('/logs', p)
  },

  traces: (params) => {
    const p = { ...params }
    if (p.from) p.from = toRFC3339(p.from)
    if (p.to) p.to = toRFC3339(p.to)
    return get('/traces', p)
  },

  trace: (id) => get(`/traces/${id}`),
}
