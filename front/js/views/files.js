const FilesView = {
  render() {
    const { parentId, breadcrumbs } = App.state;
    const bcHTML = breadcrumbs.length === 0
      ? ''
      : `<button class="topbar-back" onclick="FilesView.goUp()"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 12H5m7-7l-7 7 7 7"/></svg></button>`;
    return `
      <div class="topbar">
        ${bcHTML}
        <div class="topbar-breadcrumb" id="topbar-breadcrumb"></div>
        <div class="topbar-spacer"></div>
        <div class="topbar-actions">
          <button class="btn btn-outline" onclick="FilesView.showCreateFolder()"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14m-7-7h14"/></svg>新建文件夹</button>
          <button class="btn btn-primary" onclick="FilesView.triggerUpload()"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4m4-7l5-5 5 5m-5-5v12"/></svg>上传</button>
        </div>
        <input type="file" id="file-upload-input" style="display:none" onchange="FilesView.doUpload()" multiple>
      </div>
      <div class="content" id="files-content">
        <table class="file-table">
          <thead><tr>
            <th class="col-icon"></th>
            <th class="col-name sort-desc">名称</th>
            <th class="col-size">大小</th>
            <th class="col-type">类型</th>
            <th class="col-date">修改时间</th>
            <th class="col-actions"></th>
          </tr></thead>
          <tbody id="file-table-body"><tr><td colspan="6"><div class="empty-state"><p>加载中...</p></div></td></tr></tbody>
        </table>
      </div>`;
  },

  afterRender() {
    this.renderBreadcrumbs();
    App.updateSidebarSpace();
    return this.loadFiles();
  },

  renderBreadcrumbs() {
    const bc = App.state.breadcrumbs;
    const el = E('topbar-breadcrumb');
    if (!el) return;
    el.innerHTML = `<span onclick="FilesView.navigateTo(0, [])">📁 根目录</span>` +
      bc.map((b, i) => {
        const isLast = i === bc.length - 1;
        return `<span class="sep">›</span><span class="${isLast ? 'current' : ''}" onclick="${isLast ? '' : `FilesView.navigateTo(${b.id}, App.state.breadcrumbs.slice(0, ${i+1}))`}">${EscapeHTML(b.name)}</span>`;
      }).join('');
  },

  async loadFiles() {
    const { parentId } = App.state;
    try {
      const resp = await API.file.list(parentId);
      const list = resp.list || resp.data?.list || [];
      const tbody = E('file-table-body');
      if (list.length === 0) {
        tbody.innerHTML = `<tr><td colspan="6"><div class="empty-state"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/></svg><h3>此目录为空</h3><p>上传文件或创建文件夹</p></div></td></tr>`;
      } else {
        tbody.innerHTML = list.map(f => this.renderFileRow(f)).join('');
      }
    } catch (e) {
      E('file-table-body').innerHTML = `<tr><td colspan="6"><div class="empty-state"><p>加载失败: ${e.message}</p></div></td></tr>`;
    }
  },

  renderFileRow(f) {
    const cat = f.is_dir ? 'folder' : this.fileCategory(f.file_ext);
    const icon = f.is_dir ? '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M2 6a2 2 0 012-2h5l2 2h9a2 2 0 012 2v10a2 2 0 01-2 2H4a2 2 0 01-2-2V6z"/></svg>' : '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>';
    const info = f.is_dir ? '文件夹' : FilesView.formatSize(f.file_size);
    const safeName = EscapeHTML(f.file_name);
    const jsName = JsEscape(f.file_name);
    const time = (f.updated_at || f.created_at || '').substring(0, 16);
    return `
      <tr ondblclick="${f.is_dir ? `FilesView.enterFolder(${f.id}, '${jsName}')` : `FilesView.download(${f.id})`}" onclick="${f.is_dir ? `FilesView.enterFolder(${f.id}, '${jsName}')` : ''}">
        <td class="col-icon"><div class="file-icon-cell ${cat}">${icon}</div></td>
        <td class="col-name"><div class="file-name-cell"><span>${safeName}</span>${f.is_dir ? '<span class="file-badge">文件夹</span>' : ''}</div></td>
        <td class="col-size">${info}</td>
        <td class="col-type">${f.is_dir ? '文件夹' : (f.file_ext || '-').toUpperCase()}</td>
        <td class="col-date">${time}</td>
        <td class="col-actions" onclick="event.stopPropagation()">
          <div class="file-actions-cell">
            ${!f.is_dir ? `<button class="btn btn-ghost btn-icon" title="下载" onclick="FilesView.download(${f.id})"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4m4-7l5-5 5 5m-5-5v12"/></svg></button>` : ''}
            <button class="btn btn-ghost btn-icon" title="分享" onclick="FilesView.showShare(${f.id}, '${jsName}')"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><path d="M8.59 13.51l6.83 3.98m-.01-10.98l-6.82 3.98"/></svg></button>
            <button class="btn btn-ghost btn-icon" title="更多" onclick="FilesView.showContextMenu(event, ${f.id}, '${jsName}', ${f.is_dir})"><svg viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="5" r="2"/><circle cx="12" cy="12" r="2"/><circle cx="12" cy="19" r="2"/></svg></button>
          </div>
        </td>
      </tr>`;
  },

  fileCategory(ext) {
    const e = (ext || '').toLowerCase();
    if (['.jpg','.jpeg','.png','.gif','.webp','.svg','.bmp','.ico'].includes(e)) return 'image';
    if (['.mp4','.avi','.mov','.mkv','.webm','.flv'].includes(e)) return 'video';
    if (['.mp3','.wav','.flac','.aac','.ogg','.wma'].includes(e)) return 'audio';
    if (['.pdf','.doc','.docx','.xls','.xlsx','.ppt','.pptx','.txt','.csv','.md'].includes(e)) return 'doc';
    if (['.zip','.rar','.7z','.tar','.gz','.bz2'].includes(e)) return 'archive';
    return 'default';
  },

  formatSize(bytes) {
    if (!bytes || bytes === 0) return '-';
    const u = ['B','KB','MB','GB','TB'];
    let i = 0, s = bytes;
    while (s >= 1024 && i < u.length - 1) { s /= 1024; i++; }
    return s.toFixed(i > 0 ? 1 : 0) + ' ' + u[i];
  },

  enterFolder(id, name) {
    App.setState({ parentId: id, breadcrumbs: [...App.state.breadcrumbs, { id, name }] });
  },

  navigateTo(id, bc) {
    App.setState({ parentId: id, breadcrumbs: bc });
  },

  goUp() {
    const bc = App.state.breadcrumbs.slice(0, -1);
    App.setState({ parentId: bc.length > 0 ? bc[bc.length - 1].id : 0, breadcrumbs: bc });
  },

  triggerUpload() { E('file-upload-input').click(); },

  async doUpload() {
    const files = E('file-upload-input').files;
    if (!files.length) return;
    const { parentId } = App.state;
    App.toast(`正在上传 ${files.length} 个文件...`);
    for (const f of files) {
      try { await API.file.upload(parentId, f); } catch (e) { App.toast(`${f.name}: ${e.message}`); }
    }
    E('file-upload-input').value = '';
    App.toast('上传完成');
    this.loadFiles();
    App.updateSidebarSpace();
  },

  download(id) {
    const a = document.createElement('a');
    a.href = API.file.downloadUrl(id);
    a.download = '';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  },

  showCreateFolder() {
    Modal.show('新建文件夹', `
      <input class="form-input" id="modal-folder-name" placeholder="文件夹名称">
      <div class="modal-btns">
        <button class="btn btn-outline" onclick="Modal.hide()">取消</button>
        <button class="btn btn-primary" onclick="FilesView.doCreateFolder()">创建</button>
      </div>`);
  },

  async doCreateFolder() {
    const name = E('modal-folder-name').value.trim();
    if (!name) return App.toast('请输入名称');
    try { await API.folder.create(name, App.state.parentId); Modal.hide(); App.toast('创建成功'); this.loadFiles(); } catch (e) { App.toast(e.message); }
  },

  showShare(id, name) {
    Modal.show('创建分享', `
      <p style="margin-bottom:12px;font-size:14px;color:var(--text-secondary);">分享: ${EscapeHTML(name)}</p>
      <input class="form-input" id="modal-share-code" placeholder="提取码 (可选，留空无需验证)">
      <input class="form-input" id="modal-share-days" placeholder="有效期天数 (0=永久)" value="0" type="number" min="0">
      <div class="modal-btns">
        <button class="btn btn-outline" onclick="Modal.hide()">取消</button>
        <button class="btn btn-primary" onclick="FilesView.doCreateShare(${id})">创建分享</button>
      </div>`);
  },

  async doCreateShare(userFileId) {
    const code = E('modal-share-code').value.trim();
    const days = parseInt(E('modal-share-days').value) || 0;
    try {
      const resp = await API.share.create(userFileId, code, days);
      const d = resp.data || resp;
      const shareUrl = `${location.origin}${location.pathname}?share=${d.share_code}`;
      Modal.show('分享已创建', `
        <p style="word-break:break-all;font-size:13px;margin-bottom:8px;background:var(--bg);padding:10px;border-radius:8px;">${shareUrl}</p>
        ${d.code ? `<p style="font-size:12px;color:var(--text-muted);margin-bottom:8px;">提取码: <b>${d.code}</b></p>` : ''}
        <div class="modal-btns">
          <button class="btn btn-primary" onclick="navigator.clipboard.writeText('${shareUrl}').then(()=>{Modal.hide();App.toast('已复制')})">复制链接</button>
          <button class="btn btn-outline" onclick="Modal.hide()">关闭</button>
        </div>`);
    } catch (e) { App.toast(e.message); }
  },

  showContextMenu(e, id, name, isDir) {
    e.preventDefault();
    e.stopPropagation();
    ContextMenu.show(e.clientX, e.clientY, [
      { label: '重命名', icon: 'edit', action: () => this.showRename(id, name, isDir) },
      { label: '移入回收站', icon: 'trash', action: () => this.doTrash(id, isDir) },
      { label: '彻底删除', icon: 'delete', danger: true, action: () => this.doRemove(id, isDir) },
    ]);
  },

  showRename(id, name, isDir) {
    Modal.show('重命名', `
      <input class="form-input" id="modal-rename-input" placeholder="新名称" value="${EscapeHTML(name)}">
      <div class="modal-btns">
        <button class="btn btn-outline" onclick="Modal.hide()">取消</button>
        <button class="btn btn-primary" onclick="FilesView.doRename(${id}, ${isDir})">确认</button>
      </div>`);
  },

  async doRename(id, isDir) {
    const n = E('modal-rename-input').value.trim();
    if (!n) return App.toast('请输入名称');
    try {
      isDir ? await API.folder.rename(id, n) : await API.file.rename(id, n);
      Modal.hide(); App.toast('重命名成功'); this.loadFiles();
    } catch (e) { App.toast(e.message); }
  },

  async doTrash(id, isDir) {
    if (!confirm('移入回收站？')) return;
    try {
      isDir ? await API.folder.trash(id) : await API.file.trash(id);
      App.toast('已移入回收站'); this.loadFiles(); App.updateSidebarSpace();
    } catch (e) { App.toast(e.message); }
  },

  async doRemove(id, isDir) {
    if (!confirm('彻底删除？不可恢复！')) return;
    try {
      isDir ? await API.folder.remove(id) : await API.file.remove(id);
      App.toast('已删除'); this.loadFiles(); App.updateSidebarSpace();
    } catch (e) { App.toast(e.message); }
  },
};
