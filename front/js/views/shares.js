const SharesView = {
  render() {
    return `
      <div class="topbar">
        <div style="font-size:15px;font-weight:600;color:var(--text);">🔗 我的分享</div>
        <div class="topbar-spacer"></div>
        <button class="btn btn-outline" onclick="SharesView.refresh()">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M23 4v6h-6M1 20v-6h6"/><path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/></svg>刷新
        </button>
      </div>
      <div class="content" id="shares-content">
        <div class="share-grid" id="shares-grid"></div>
      </div>`;
  },

  afterRender() { return this.load(); },

  async load() {
    try {
      const resp = await API.share.list();
      const list = resp.list || resp.data?.list || [];
      const content = E('shares-content');
      if (list.length === 0) {
        content.innerHTML = `<div style="flex:1;display:flex;align-items:center;justify-content:center;"><div class="empty-state"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><path d="M8.59 13.51l6.83 3.98m-.01-10.98l-6.82 3.98"/></svg><h3>暂无分享</h3><p>在文件列表中点击分享按钮创建</p></div></div>`;
      } else {
        content.innerHTML = `<div class="share-grid">${list.map(s => this.renderCard(s)).join('')}</div>`;
      }
    } catch (e) { E('shares-content').innerHTML = `<div style="flex:1;display:flex;align-items:center;justify-content:center;"><div class="empty-state"><p>加载失败</p></div></div>`; }
  },

  renderCard(s) {
    const url = `${location.origin}${location.pathname}?share=${s.share_code}`;
    const expireText = s.expire_at ? '至 ' + new Date(s.expire_at).toLocaleDateString() : '永久有效';
    return `
      <div class="share-card">
        <div class="share-card-info">
          <div class="share-card-name">${EscapeHTML(s.file_name)}</div>
          <div class="share-card-meta">
            <span>👁 ${s.view_count || 0} 次</span>
            <span>${s.code ? '🔒 需提取码' : '🌐 公开'}</span>
            <span>⏰ ${expireText}</span>
          </div>
        </div>
        <div class="share-card-actions">
          <button class="btn btn-outline btn-icon" title="复制链接" onclick="navigator.clipboard.writeText('${url}').then(()=>App.toast('已复制'))"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg></button>
          <button class="btn btn-danger-ghost btn-icon" title="取消分享" onclick="SharesView.revoke('${s.share_code}')"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 01-2 2H8a2 2 0 01-2-2L5 6"/></svg></button>
        </div>
      </div>`;
  },

  async revoke(code) {
    if (!confirm('取消此分享？')) return;
    try { await API.share.revoke(code); App.toast('已取消'); this.load(); } catch (e) { App.toast(e.message); }
  },

  refresh() { this.load(); },
};
