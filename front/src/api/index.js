const BASE = '/api/v1'

function getAccessToken() {
  return localStorage.getItem('access_token')
}
function getRefreshToken() {
  return localStorage.getItem('refresh_token')
}
export function setTokens(access, refresh) {
  localStorage.setItem('access_token', access)
  localStorage.setItem('refresh_token', refresh)
}
export function clearTokens() {
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
}
export function isAuthenticated() {
  return !!getAccessToken()
}

async function request(method, path, body = null, multipart = false) {
  const url = BASE + path
  const headers = {}

  const token = getAccessToken()
  if (token) {
    headers['Authorization'] = 'Bearer ' + token
  }

  if (!multipart && body) {
    headers['Content-Type'] = 'application/json'
  }

  const opts = { method, headers }
  if (body) {
    opts.body = multipart ? body : JSON.stringify(body)
  }

  let res = await fetch(url, opts)

  if (res.status === 401 && getRefreshToken()) {
    const refreshed = await refreshAccessToken()
    if (refreshed) {
      headers['Authorization'] = 'Bearer ' + getAccessToken()
      opts.headers = headers
      res = await fetch(url, opts)
    } else if (!getAccessToken()) {
      window.location.hash = '#/login'
      throw new Error('AUTH_EXPIRED')
    }
  }

  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(data.error || data.msg || `HTTP ${res.status}`)
  }
  return data.data !== undefined ? data.data : data
}

let _refreshPromise = null

async function refreshAccessToken() {
  if (_refreshPromise) return _refreshPromise
  _refreshPromise = _doRefresh()
  try {
    return await _refreshPromise
  } finally {
    _refreshPromise = null
  }
}

async function _doRefresh() {
  const currentRefresh = getRefreshToken()
  if (!currentRefresh) {
    clearTokens()
    return false
  }
  try {
    const res = await fetch(BASE + '/user/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: currentRefresh }),
    })
    const data = await res.json()
    if (res.ok && data.data && data.data.access_token) {
      setTokens(data.data.access_token, currentRefresh)
      return true
    }
    if (res.status === 401 || res.status === 403) {
      clearTokens()
    }
    return false
  } catch (e) {
    return false
  }
}

function getQuery(params = {}) {
  const qs = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== '') qs.set(k, v)
  })
  const s = qs.toString()
  return s ? '?' + s : ''
}

// ========== User ==========
export const userApi = {
  register(body) {
    return request('POST', '/user/register', body)
  },
  login(body) {
    return request('POST', '/user/login', body)
  },
  getInfo() {
    return request('GET', '/user/info')
  },
  updateInfo(body) {
    return request('PUT', '/user/info', body)
  },
  getSpace() {
    return request('GET', '/user/space')
  },
}

// ========== File ==========
export const fileApi = {
  upload(formData) {
    return request('POST', '/file/upload', formData, true)
  },
  download(userfileId) {
    const token = getAccessToken()
    return BASE + '/file/download/' + userfileId + '?token=' + encodeURIComponent(token)
  },
  list(params) {
    return request('GET', '/file/list' + getQuery(params))
  },
  rename(body) {
    return request('PUT', '/file/rename', body)
  },
  move(body) {
    return request('PUT', '/file/move', body)
  },
  toTrash(userfileId) {
    return request('DELETE', '/file/delete/' + userfileId)
  },
  remove(userfileId) {
    return request('DELETE', '/file/remove/' + userfileId)
  },
}

// ========== Folder ==========
export const folderApi = {
  create(body) {
    return request('POST', '/folder/create', body)
  },
  rename(body) {
    return request('PUT', '/folder/rename', body)
  },
  move(body) {
    return request('PUT', '/folder/move', body)
  },
  toTrash(userfolderId) {
    return request('DELETE', '/folder/delete/' + userfolderId)
  },
  remove(userfolderId) {
    return request('DELETE', '/folder/remove/' + userfolderId)
  },
}

// ========== Trash ==========
export const trashApi = {
  list(params) {
    return request('GET', '/trash/list' + getQuery(params))
  },
  restoreFile(userfileId) {
    return request('POST', '/trash/file/' + userfileId)
  },
  restoreFolder(userfolderId) {
    return request('POST', '/trash/folder/' + userfolderId)
  },
}

// ========== Chunk Upload ==========
export const chunkApi = {
  init(body) {
    return request('POST', '/file/chunk/init', body)
  },
  upload(formData) {
    return request('POST', '/file/chunk/upload', formData, true)
  },
  complete(body) {
    return request('POST', '/file/chunk/complete', body)
  },
  status(uploadId) {
    return request('GET', '/file/chunk/status?upload_id=' + uploadId)
  },
}

// ========== Share ==========
export const shareApi = {
  create(body) {
    return request('POST', '/share/create', body)
  },
  list(params) {
    return request('GET', '/share/list' + getQuery(params))
  },
  revoke(shareCode) {
    return request('DELETE', '/share/' + shareCode)
  },
  getInfo(shareCode, extractionCode) {
    const params = new URLSearchParams()
    if (extractionCode) params.set('code', extractionCode)
    return request('GET', '/share/' + shareCode + '/info?' + params.toString())
  },
  download(shareCode, extractionCode) {
    const params = new URLSearchParams()
    if (extractionCode) params.set('code', extractionCode)
    return BASE + '/share/' + shareCode + '/download?' + params.toString()
  },
}

// ========== Task ==========
export const taskApi = {
  create(body) {
    return request('POST', '/task/create', body)
  },
  upload(taskId, fileIndex, formData) {
    return request('POST', '/task/' + taskId + '/file', formData, true)
  },
  progress(taskId) {
    return request('GET', '/task/' + taskId + '/progress')
  },
}
