const UNITS = ['B', 'KB', 'MB', 'GB', 'TB']

export function formatSize(bytes) {
  if (bytes === 0) return '0 B'
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), UNITS.length - 1)
  return (bytes / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 2) + ' ' + UNITS[i]
}

export function formatDate(dateStr) {
  if (!dateStr) return '--'
  const d = new Date(dateStr)
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const h = String(d.getHours()).padStart(2, '0')
  const min = String(d.getMinutes()).padStart(2, '0')
  return `${y}-${m}-${day} ${h}:${min}`
}

export function formatChunkProgress(completed, total) {
  const pct = total > 0 ? Math.round((completed / total) * 100) : 0
  const barLen = 20
  const filledLen = Math.round((pct / 100) * barLen)
  return `[${'='.repeat(filledLen)}${'·'.repeat(barLen - filledLen)}] ${pct}%`
}

export function truncate(str, n = 32) {
  if (!str) return ''
  return str.length > n ? str.slice(0, n - 3) + '…' : str
}
