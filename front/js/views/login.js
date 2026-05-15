const LoginView = {
  render() {
    const isRegister = App.state.loginMode === 'register';
    return `
      <div class="login-page">
        <div class="login-card">
          <h1>GoNetDisk</h1>
          <div class="subtitle">${isRegister ? '创建账号，开始使用' : '登录您的网盘'}</div>
          ${isRegister ? '<div class="form-group"><label>用户名</label><input class="form-input" id="reg-username" type="text" placeholder="设置用户名" autocomplete="username"></div>' : ''}
          <div class="form-group"><label>邮箱</label><input class="form-input" id="reg-email" type="email" placeholder="your@email.com" autocomplete="email" value="${EscapeHTML(localStorage.getItem('gonetdisk_email') || '')}"></div>
          <div class="form-group"><label>密码</label><input class="form-input" id="reg-password" type="password" placeholder="${isRegister ? '设置密码' : '输入密码'}" autocomplete="${isRegister ? 'new-password' : 'current-password'}"></div>
          ${isRegister ? '<div class="form-group"><label>确认密码</label><input class="form-input" id="reg-password2" type="password" placeholder="再次输入密码" autocomplete="new-password"></div>' : ''}
          <button class="btn btn-primary" onclick="LoginView.submit()">${isRegister ? '注册' : '登录'}</button>
          <span class="toggle-link" onclick="App.setState({loginMode: ${isRegister} ? 'login' : 'register'})">
            ${isRegister ? '已有账号？去登录' : '没有账号？去注册'}
          </span>
        </div>
      </div>`;
  },

  async submit() {
    try {
      const emailEl = E('reg-email');
      const passwordEl = E('reg-password');
      if (!emailEl || !passwordEl) { App.toast('页面加载异常，请刷新'); return; }

      const email = emailEl.value.trim();
      const password = passwordEl.value;
      const isRegister = App.state.loginMode === 'register';
      if (!email || !password) { App.toast('请填写完整信息'); return; }

      let resp;
      if (isRegister) {
        const usernameEl = E('reg-username');
        const password2El = E('reg-password2');
        if (!usernameEl || !password2El) { App.toast('页面加载异常，请刷新'); return; }
        const username = usernameEl.value.trim();
        const password2 = password2El.value;
        if (!username) { App.toast('请输入用户名'); return; }
        if (password !== password2) { App.toast('两次密码不一致'); return; }
        resp = await API.auth.register(email, username, password);
      } else {
        resp = await API.auth.login(email, password);
      }
      if (resp.data?.token) API.setToken(resp.data.token);
      localStorage.setItem('gonetdisk_email', email);
      localStorage.setItem('gonetdisk_username', resp.data?.username || resp.username || '');
      App.setState({ currentView: 'files', loginMode: 'login' });
      App.toast(isRegister ? '注册成功' : '登录成功');
    } catch (e) { App.toast(e.message); }
  },
};
