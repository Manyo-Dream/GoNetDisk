const App = {
  state: {
    currentView: 'login',
    loginMode: 'login',
    parentId: 0,
    breadcrumbs: [],
  },

  views: { login: LoginView, files: FilesView, trash: TrashView, shares: SharesView },

  async init() {
    if (API.hasToken()) {
      try { await API.auth.info(); this.state.currentView = 'files'; } catch (e) { API.clearToken(); this.state.currentView = 'login'; }
    }
    try {
      const shareCode = new URLSearchParams(location.search).get('share');
      if (shareCode) { this.showSharedItem(shareCode); return; }
    } catch (e) {}
    this.render();
  },

  setState(update) {
    Object.assign(this.state, update);
    this.render();
  },

  render() {
    try {
      const viewName = this.state.currentView;
      const view = this.views[viewName];
      if (!view || !view.render) return;

      if (viewName === 'login') {
        const mainEl = E('app-main-layout');
        const loginEl = E('app-login-page');
        if (mainEl) mainEl.classList.remove('active');
        if (loginEl) {
          loginEl.classList.add('active');
          loginEl.innerHTML = view.render();
        }
      } else {
        const loginEl = E('app-login-page');
        const mainEl = E('app-main-layout');
        const contentEl = E('main-content-area');
        if (loginEl) loginEl.classList.remove('active');
        if (mainEl) mainEl.classList.add('active');
        if (contentEl) contentEl.innerHTML = view.render();
        this.updateSidebar();
        if (view.afterRender) {
          const p = view.afterRender();
          if (p && typeof p.catch === 'function') p.catch(function() {});
        }
      }
    } catch (e) {
      console.error('render error:', e);
      alert('页面渲染错误: ' + e.message);
    }
  },

  updateSidebar() {
    const items = document.querySelectorAll('.sidebar-item');
    items.forEach(el => el.classList.toggle('active', el.dataset.view === this.state.currentView));
  },

  async updateSidebarSpace() {
    try {
      const resp = await API.auth.space();
      const d = resp.data || resp;
      const pct = d.total_space > 0 ? (d.used_space / d.total_space * 100).toFixed(1) : 0;
      E('sidebar-storage-fill').style.width = Math.min(pct, 100) + '%';
      E('sidebar-storage-text').textContent = `已用 ${FilesView.formatSize(d.used_space)} / ${FilesView.formatSize(d.total_space)}`;
    } catch (e) {}
  },

  async showSpace() {
    try {
      const resp = await API.auth.space();
      const d = resp.data || resp;
      Modal.show('空间信息', `
        <div style="text-align:center;padding:10px 0;">
          <div style="font-size:28px;font-weight:700;">${FilesView.formatSize(d.used_space)}</div>
          <div style="font-size:14px;color:var(--text-muted);margin:8px 0;">已使用 / 总计 ${FilesView.formatSize(d.total_space)}</div>
          <div class="sidebar-storage-bar"><div class="sidebar-storage-fill" style="width:${d.total_space > 0 ? Math.min(d.used_space / d.total_space * 100, 100) : 0}%"></div></div>
          <div style="font-size:13px;color:var(--text-muted);margin-top:4px;">剩余 ${FilesView.formatSize(d.total_space - d.used_space)}</div>
        </div>
        <div class="modal-btns"><button class="btn btn-primary" onclick="Modal.hide()">关闭</button></div>`);
    } catch (e) { App.toast(e.message); }
  },

  async showSharedItem(code) {
    const pw = prompt('请输入提取码 (无提取码请留空):');
    if (pw === null) { location.href = location.pathname; return; }
    try {
      const resp = await API.share.info(code, pw || '');
      const d = resp.data || resp;
      if (!d.is_dir) { window.location.href = API.share.downloadUrl(code, pw || ''); }
      else App.toast('文件夹暂不支持在线预览');
    } catch (e) { alert(e.message); location.href = location.pathname; }
  },

  async logout() {
    if (!confirm('确定要退出登录吗？')) return;
    API.clearToken();
    this.setState({ currentView: 'login', loginMode: 'login', parentId: 0, breadcrumbs: [] });
  },

  toast(msg) {
    const existing = document.querySelector('.toast');
    if (existing) existing.remove();
    const el = document.createElement('div');
    el.className = 'toast';
    el.textContent = msg;
    document.body.appendChild(el);
    setTimeout(() => el.remove(), 2500);
  },
};

const Modal = {
  show(title, body) {
    this.hide();
    const el = document.createElement('div');
    el.className = 'modal-overlay';
    el.id = 'modal-overlay';
    el.innerHTML = `<div class="modal"><h3>${title}</h3>${body}</div>`;
    el.addEventListener('click', (e) => { if (e.target === el) this.hide(); });
    document.body.appendChild(el);
  },
  hide() { const el = document.getElementById('modal-overlay'); if (el) el.remove(); },
};

const ContextMenu = {
  show(x, y, items) {
    this.hide();
    const el = document.createElement('div');
    el.className = 'context-menu';
    el.id = 'context-menu';
    el.style.left = Math.min(x, window.innerWidth - 200) + 'px';
    el.style.top = Math.min(y, window.innerHeight - items.length * 40 - 20) + 'px';
    el.innerHTML = items.map((item, i) => {
      const iconMap = { edit: '<path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/>', trash: '<polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 01-2 2H8a2 2 0 01-2-2L5 6m5 0V4a1 1 0 011-1h2a1 1 0 011 1v2"/>', delete: '<polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 01-2 2H8a2 2 0 01-2-2L5 6"/>' };
      return `<div class="context-menu-item${item.danger ? ' danger' : ''}" data-idx="${i}"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">${iconMap[item.icon] || ''}</svg>${item.label}</div>`;
    }).join('');
    el.addEventListener('click', (e) => {
      const idx = e.target.closest('.context-menu-item')?.dataset?.idx;
      if (idx !== undefined) {
        e.stopPropagation();
        this.hide();
        items[parseInt(idx)].action();
      }
    });
    document.body.appendChild(el);
    setTimeout(() => document.addEventListener('click', this.hide, { once: true }), 0);
  },
  hide() { const el = document.getElementById('context-menu'); if (el) el.remove(); },
};

function E(id) { return document.getElementById(id); }

document.addEventListener('DOMContentLoaded', () => App.init());
document.addEventListener('click', (e) => {
  const menu = document.getElementById('context-menu');
  if (menu && !menu.contains(e.target)) ContextMenu.hide();
});
