function EscapeHTML(s) { return (s || '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;'); }
function JsEscape(s)   { return (s || '').replace(/\\/g,'\\\\').replace(/'/g,"\\'").replace(/\n/g,'\\n'); }

const API = (() => {
  const BASE = '/api/v1';
  let token = localStorage.getItem('gonetdisk_token') || '';

  function headers() {
    const h = { 'Content-Type': 'application/json' };
    if (token) h['Authorization'] = 'Bearer ' + token;
    return h;
  }

  function headersMultipart() {
    const h = {};
    if (token) h['Authorization'] = 'Bearer ' + token;
    return h;
  }

  async function request(method, url, body, isJson = true) {
    const opts = { method, headers: isJson ? headers() : headersMultipart() };
    if (body && isJson) opts.body = JSON.stringify(body);
    if (body && !isJson) opts.body = body;
    const res = await fetch(BASE + url, opts);
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || '请求失败');
    return data;
  }

  function setToken(t) { token = t; localStorage.setItem('gonetdisk_token', token); }
  function clearToken() { token = ''; localStorage.removeItem('gonetdisk_token'); }
  function hasToken() { return !!token; }

  return {
    setToken, clearToken, hasToken,

    auth: {
      register: (email, username, password) =>
        request('POST', '/user/register', { email, username, password }),
      login: (email, password) =>
        request('POST', '/user/login', { email, password }),
      info: () => request('GET', '/user/info'),
      updateInfo: (data) => request('PUT', '/user/info', data),
      space: () => request('GET', '/user/space'),
    },

    file: {
      list: (parentId, page, pageSize, sortBy, orderBy) =>
        request('GET', `/file/list?parent_id=${parentId || 0}&page=${page || 1}&page_size=${pageSize || 20}&sort_by=${sortBy || 'updated_at'}&order_by=${orderBy || 'desc'}`),
      upload: (parentId, file) => {
        const fd = new FormData();
        fd.append('parent_id', parentId || 0);
        fd.append('file', file);
        return request('POST', '/file/upload', fd, false);
      },
      downloadUrl: (fileId) => `${BASE}/file/download/${fileId}?token=${encodeURIComponent(token)}`,
      rename: (userFileId, newFileName) =>
        request('PUT', '/file/rename', { user_file_id: userFileId, new_file_name: newFileName }),
      move: (userFileId, targetParentId) =>
        request('PUT', '/file/move', { user_file_id: userFileId, target_parent_id: targetParentId }),
      trash: (userFileId) =>
        request('DELETE', `/file/delete/${userFileId}`),
      remove: (userFileId) =>
        request('DELETE', `/file/remove/${userFileId}`),
      restore: (userFileId) =>
        request('POST', `/trash/file/${userFileId}`),
    },

    folder: {
      create: (folderName, parentId) =>
        request('POST', '/folder/create', { folder_name: folderName, parent_id: parentId || 0 }),
      trash: (folderId) =>
        request('DELETE', `/folder/delete/${folderId}`),
      remove: (folderId) =>
        request('DELETE', `/folder/remove/${folderId}`),
      restore: (folderId) =>
        request('POST', `/trash/folder/${folderId}`),
      rename: (userFolderId, newFolderName) =>
        request('PUT', '/folder/rename', { user_folder_id: userFolderId, new_folder_name: newFolderName }),
      move: (userFolderId, targetParentId) =>
        request('PUT', '/folder/move', { user_folder_id: userFolderId, target_parent_id: targetParentId }),
    },

    trash: {
      list: (page, pageSize) =>
        request('GET', `/trash/list?page=${page || 1}&page_size=${pageSize || 20}`),
    },

    share: {
      create: (userFileId, code, expireDays) =>
        request('POST', '/share/create', { user_file_id: userFileId, code: code || '', expire_days: expireDays || 0 }),
      list: (page, pageSize) =>
        request('GET', `/share/list?page=${page || 1}&page_size=${pageSize || 20}`),
      revoke: (shareCode) =>
        request('DELETE', `/share/${shareCode}`),
      info: (shareCode, code) =>
        request('GET', `/share/${shareCode}/info?code=${encodeURIComponent(code || '')}`),
      downloadUrl: (shareCode, code) =>
        `${BASE}/share/${shareCode}/download?code=${encodeURIComponent(code || '')}`,
    },
  };
})();
