<script setup>
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { userApi, setTokens } from '../api/index.js'
import { useI18n } from '../i18n/index.js'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const username = ref('')
const email = ref('')
const password = ref('')
const confirm = ref('')
const error = ref('')
const loading = ref(false)

async function handleRegister() {
  error.value = ''
  if (!username.value || !email.value || !password.value) {
    error.value = t('auth.allFieldsRequired')
    return
  }
  if (password.value !== confirm.value) {
    error.value = t('auth.passwordsNotMatch')
    return
  }
  if (password.value.length < 6) {
    error.value = t('auth.passwordTooShort')
    return
  }
  if (!/[a-zA-Z]/.test(password.value) || !/[0-9]/.test(password.value)) {
    error.value = t('auth.passwordNoLetterOrDigit')
    return
  }
  loading.value = true
  try {
    const data = await userApi.register({
      username: username.value,
      email: email.value,
      password: password.value,
    })
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
        <h1>{{ t('auth.register') }}</h1>
        <p>{{ t('auth.createAccount') }}</p>
      </div>

      <form @submit.prevent="handleRegister" class="auth-form">
        <label>
          <span>{{ t('auth.username') }}</span>
          <input v-model="username" type="text" name="username" :placeholder="t('auth.usernamePlaceholder')" autocomplete="username" />
        </label>
        <label>
          <span>{{ t('auth.email') }}</span>
          <input v-model="email" type="email" name="email" :placeholder="t('auth.emailPlaceholder')" autocomplete="email" spellcheck="false" />
        </label>
        <label>
          <span>{{ t('auth.password') }}</span>
          <input v-model="password" type="password" name="password" :placeholder="t('auth.newPasswordPlaceholder')" autocomplete="new-password" />
        </label>
        <label>
          <span>{{ t('auth.confirmPassword') }}</span>
          <input v-model="confirm" type="password" name="confirm" :placeholder="t('auth.confirmPlaceholder')" autocomplete="new-password" />
        </label>

        <div v-if="error" class="error-msg">! {{ error }}</div>

        <button type="submit" class="btn btn-primary" :disabled="loading">
          <span v-if="loading" class="cursor-blink">_</span>
          <span v-else>[+] {{ t('auth.registerBtn') }}</span>
        </button>
      </form>

      <div class="card-footer">
        <router-link to="/login">[<] {{ t('auth.backToLogin') }}</router-link>
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
