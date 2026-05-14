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
          <span class="toggle-link" onclick="App.setState({loginMode: isRegister ? 'login' : 'register'})">
            ${isRegister ? '已有账号？去登录' : '没有账号？去注册'}
          </span>
        </div>
      </div>`;
  },

  async submit() {
    const email = E('reg-email').value.trim();
    const password = E('reg-password').value;
    const isRegister = App.state.loginMode === 'register';
    if (!email || !password) return App.toast('请填写完整信息');
    try {
      let resp;
      if (isRegister) {
        const username = E('reg-username').value.trim();
        const password2 = E('reg-password2').value;
        if (!username) return App.toast('请输入用户名');
        if (password !== password2) return App.toast('两次密码不一致');
        resp = await API.auth.register(email, username, password);
      } else {
        resp = await API.auth.login(email, password);
      }
      if (resp.token) API.setToken(resp.token);
      localStorage.setItem('gonetdisk_email', email);
      localStorage.setItem('gonetdisk_username', resp.username || '');
      App.setState({ currentView: 'files', loginMode: 'login' });
      App.toast(isRegister ? '注册成功' : '登录成功');
    } catch (e) { App.toast(e.message); }
  },
};
