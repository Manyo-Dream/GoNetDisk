const TrashView = {
  render() {
    return `
      <div class="topbar">
        <div style="font-size:15px;font-weight:600;color:var(--text);">🗑 回收站</div>
        <div class="topbar-spacer"></div>
        <button class="btn btn-outline" onclick="TrashView.refresh()">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M23 4v6h-6M1 20v-6h6"/><path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/></svg>刷新
        </button>
      </div>
      <div class="content">
        <table class="file-table">
          <thead><tr><th class="col-icon"></th><th class="col-name">名称</th><th class="col-size">大小</th><th class="col-date">删除时间</th><th class="col-actions"></th></tr></thead>
          <tbody id="trash-table-body"><tr><td colspan="5"><div class="empty-state"><p>加载中...</p></div></td></tr></tbody>
        </table>
      </div>`;
  },

  afterRender() { return this.load(); },

  async load() {
    try {
      const resp = await API.trash.list();
      const list = resp.list || resp.data?.list || [];
      const tbody = E('trash-table-body');
      if (list.length === 0) {
        tbody.innerHTML = `<tr><td colspan="5"><div class="empty-state"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M3 6h18M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2m3 0v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6h14"/></svg><h3>回收站为空</h3></div></td></tr>`;
      } else {
        tbody.innerHTML = list.map(f => this.renderRow(f)).join('');
      }
    } catch (e) {
      E('trash-table-body').innerHTML = `<tr><td colspan="5"><div class="empty-state"><p>加载失败</p></div></td></tr>`;
    }
  },

  renderRow(f) {
    const safeName = EscapeHTML(f.file_name);
    const time = (f.deleted_at || '').substring(0, 16);
    return `
      <tr>
        <td class="col-icon"><div class="file-icon-cell ${f.is_dir ? 'folder' : 'default'}">${f.is_dir ? '📁' : '📄'}</div></td>
        <td class="col-name"><div class="file-name-cell"><span>${safeName}</span></div></td>
        <td class="col-size">${FilesView.formatSize(f.file_size)}</td>
        <td class="col-date">${time}</td>
        <td class="col-actions" onclick="event.stopPropagation()">
          <div class="file-actions-cell">
            <button class="btn btn-ghost btn-icon" title="还原" onclick="TrashView.restore(${f.id}, ${f.is_dir})"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 12a9 9 0 119 9"/><path d="M3 3v6h6m5-5l-3 3 3 3"/></svg></button>
            <button class="btn btn-danger-ghost btn-icon" title="彻底删除" onclick="TrashView.remove(${f.id}, ${f.is_dir})"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 01-2 2H8a2 2 0 01-2-2L5 6m5 0V4a1 1 0 011-1h2a1 1 0 011 1v2"/></svg></button>
          </div>
        </td>
      </tr>`;
  },

  async restore(id, isDir) {
    try {
      isDir ? await API.folder.restore(id) : await API.file.restore(id);
      App.toast('已还原'); this.load(); App.updateSidebarSpace();
    } catch (e) { App.toast(e.message); }
  },

  async remove(id, isDir) {
    if (!confirm('彻底删除？不可恢复！')) return;
    try {
      isDir ? await API.folder.remove(id) : await API.file.remove(id);
      App.toast('已删除'); this.load(); App.updateSidebarSpace();
    } catch (e) { App.toast(e.message); }
  },

  refresh() { this.load(); },
};
