<script setup>
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { userApi, setTokens, isAuthenticated } from '../api/index.js'
import { useI18n } from '../i18n/index.js'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function handleLogin() {
  error.value = ''
  if (!email.value || !password.value) {
    error.value = t('auth.allFieldsRequired')
    return
  }
  loading.value = true
  try {
    const data = await userApi.login({ email: email.value, password: password.value })
    setTokens(data.access_token, data.refresh_token)
    const redirect = route.query.redirect
    router.push(redirect || '/files')
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-card">
      <div class="card-header">
        <span class="logo">[gnd]</span>
        <h1>GoNetDisk</h1>
        <p>{{ t('auth.authenticationRequired') }}</p>
      </div>

      <form @submit.prevent="handleLogin" class="auth-form">
        <label>
          <span>{{ t('auth.email') }}</span>
          <input v-model="email" type="email" name="email" :placeholder="t('auth.emailPlaceholder')" autocomplete="email" spellcheck="false" />
        </label>
        <label>
          <span>{{ t('auth.password') }}</span>
          <input v-model="password" type="password" name="password" :placeholder="t('auth.passwordPlaceholder')" autocomplete="current-password" />
        </label>

        <div v-if="error" class="error-msg">! {{ error }}</div>

        <button type="submit" class="btn btn-primary" :disabled="loading">
          <span v-if="loading" class="cursor-blink">_</span>
          <span v-else>[>] {{ t('auth.loginBtn') }}</span>
        </button>
      </form>

      <div class="card-footer">
        <router-link to="/register">[+] {{ t('auth.createAccount') }}</router-link>
      </div>
    </div>
  </div>
</template>

<style scoped>
.auth-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100%;
  padding: var(--gap-6);
}
.auth-card {
  width: 380px;
  background: var(--bg-surface);
  border: 1px solid var(--border);
}
.card-header {
  text-align: center;
  padding: var(--gap-8) var(--gap-6) var(--gap-6);
  border-bottom: 1px solid var(--border-faint);
}
.logo {
  display: block;
  font-size: var(--fs-xl);
  color: var(--accent);
  font-weight: 700;
  margin-bottom: var(--gap-2);
}
h1 {
  font-size: var(--fs-lg);
  font-weight: 600;
  margin-bottom: var(--gap-1);
}
p {
  font-size: var(--fs-xs);
  color: var(--text-dim);
  letter-spacing: 0.08em;
}
.auth-form {
  display: flex;
  flex-direction: column;
  gap: var(--gap-4);
  padding: var(--gap-6);
}
label {
  display: flex;
  flex-direction: column;
  gap: var(--gap-1);
}
label span {
  font-size: var(--fs-xs);
  color: var(--text-dim);
  font-weight: 600;
  letter-spacing: 0.06em;
}
label input {
  padding: var(--gap-3);
}
.error-msg {
  color: var(--red);
  font-size: var(--fs-xs);
  padding: var(--gap-2);
  border: 1px solid var(--red-dim);
  background: var(--red-dim);
}
.card-footer {
  text-align: center;
  padding: var(--gap-4) var(--gap-6);
  border-top: 1px solid var(--border-faint);
  font-size: var(--fs-xs);
}
</style>
