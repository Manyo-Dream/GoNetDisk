const BASE = "/api/v1";

function getAccessToken() {
  return localStorage.getItem("access_token");
}
function getRefreshToken() {
  return localStorage.getItem("refresh_token");
}
export function setTokens(access, refresh) {
  localStorage.setItem("access_token", access);
  localStorage.setItem("refresh_token", refresh);
}
export function clearTokens() {
  localStorage.removeItem("access_token");
  localStorage.removeItem("refresh_token");
}
export function isAuthenticated() {
  return !!getAccessToken();
}

async function request(method, path, body = null, multipart = false) {
  const url = BASE + path;
  const headers = {};

  const token = getAccessToken();
  if (token) {
    headers["Authorization"] = "Bearer " + token;
  }

  if (!multipart && body) {
    headers["Content-Type"] = "application/json";
  }

  const opts = { method, headers };
  if (body) {
    opts.body = multipart ? body : JSON.stringify(body);
  }

  let res = await fetch(url, opts);

  if (res.status === 401 && getRefreshToken()) {
    const refreshed = await refreshAccessToken();
    if (refreshed) {
      headers["Authorization"] = "Bearer " + getAccessToken();
      opts.headers = headers;
      res = await fetch(url, opts);
    } else if (!getAccessToken()) {
      window.location.hash = "#/login";
      throw new Error("AUTH_EXPIRED");
    }
  }

  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error || data.msg || `HTTP ${res.status}`);
  }
  return data.data !== undefined ? data.data : data;
}

let _refreshPromise = null;

async function refreshAccessToken() {
  if (_refreshPromise) return _refreshPromise;
  _refreshPromise = _doRefresh();
  try {
    return await _refreshPromise;
  } finally {
    _refreshPromise = null;
  }
}

async function _doRefresh() {
  const currentRefresh = getRefreshToken();
  if (!currentRefresh) {
    clearTokens();
    return false;
  }
  try {
    const res = await fetch(BASE + "/user/refresh", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: currentRefresh }),
    });
    const data = await res.json();
    if (res.ok && data.data && data.data.access_token) {
      setTokens(data.data.access_token, currentRefresh);
      return true;
    }
    if (res.status === 401 || res.status === 403) {
      clearTokens();
    }
    return false;
  } catch (e) {
    return false;
  }
}

function getQuery(params = {}) {
  const qs = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== "") qs.set(k, v);
  });
  const s = qs.toString();
  return s ? "?" + s : "";
}

// ========== User ==========
export const userApi = {
  register(body) {
    return request("POST", "/user/register", body);
  },
  login(body) {
    return request("POST", "/user/login", body);
  },
  getInfo() {
    return request("GET", "/user/info");
  },
  updateInfo(body) {
    return request("PUT", "/user/info", body);
  },
  getSpace() {
    return request("GET", "/user/space");
  },
};

// ========== File ==========
export const fileApi = {
  upload(formData) {
    return request("POST", "/files", formData, true);
  },
  download(userfileId) {
    return BASE + "/files/" + userfileId;
  },
  list(params) {
    return request("GET", "/files" + getQuery(params));
  },
  rename(body) {
    return request("PUT", "/files/" + body.user_file_id, body);
  },
  move(body) {
    return request("PATCH", "/files/" + body.user_file_id, body);
  },
  toTrash(userfileId) {
    return request("DELETE", "/files/" + userfileId);
  },
};

// ========== Folder ==========
export const folderApi = {
  create(body) {
    return request("POST", "/folders", body);
  },
  rename(body) {
    return request("PUT", "/folders/" + body.user_folder_id, body);
  },
  move(body) {
    return request("PATCH", "/folders/" + body.user_folder_id, body);
  },
  toTrash(userfolderId) {
    return request("DELETE", "/folders/" + userfolderId);
  },
};

// ========== Trash ==========
export const trashApi = {
  list(params) {
    return request("GET", "/trash" + getQuery(params));
  },
  restoreFile(userfileId) {
    return request("POST", "/trash/files/" + userfileId + "/restore");
  },
  restoreFolder(userfolderId) {
    return request("POST", "/trash/folders/" + userfolderId + "/restore");
  },
  removeFile(userfileId) {
    return request("DELETE", "/trash/files/" + userfileId);
  },
  removeFolder(userfolderId) {
    return request("DELETE", "/trash/folders/" + userfolderId);
  },
};

// ========== Chunk Upload ==========
export const chunkApi = {
  init(body) {
    return request("POST", "/files/chunks", body);
  },
  upload(formData) {
    return request("PUT", "/files/chunks", formData, true);
  },
  complete(body) {
    return request("POST", "/files/chunks/complete", body);
  },
  status(uploadId) {
    return request("GET", "/files/chunks/status?upload_id=" + uploadId);
  },
};

// ========== Share ==========
export const shareApi = {
  create(body) {
    return request("POST", "/shares", body);
  },
  list(params) {
    return request("GET", "/shares" + getQuery(params));
  },
  revoke(shareCode) {
    return request("DELETE", "/shares/" + shareCode);
  },
  getInfo(shareCode, extractionCode) {
    const params = new URLSearchParams();
    if (extractionCode) params.set("code", extractionCode);
    return request("GET", "/shares/" + shareCode + "?" + params.toString());
  },
  download(shareCode, extractionCode) {
    const params = new URLSearchParams();
    if (extractionCode) params.set("code", extractionCode);
    return BASE + "/shares/" + shareCode + "/download?" + params.toString();
  },
};

// ========== Task ==========
export const taskApi = {
  create(body) {
    return request("POST", "/tasks", body);
  },
  upload(taskId, fileIndex, formData) {
    return request("POST", "/tasks/" + taskId + "/files", formData, true);
  },
  progress(taskId) {
    return request("GET", "/tasks/" + taskId);
  },
};
