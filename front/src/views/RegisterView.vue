<script setup>
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { userApi, setTokens } from '../api/index.js'

const router = useRouter()
const route = useRoute()
const username = ref('')
const email = ref('')
const password = ref('')
const confirm = ref('')
const error = ref('')
const loading = ref(false)

async function handleRegister() {
  error.value = ''
  if (!username.value || !email.value || !password.value) {
    error.value = 'ALL FIELDS REQUIRED'
    return
  }
  if (password.value !== confirm.value) {
    error.value = 'PASSPHRASES DO NOT MATCH'
    return
  }
  if (password.value.length < 6) {
    error.value = 'PASSPHRASE TOO SHORT (MIN 6)'
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
        <h1>REGISTER</h1>
        <p>CREATE NEW ACCOUNT</p>
      </div>

      <form @submit.prevent="handleRegister" class="auth-form">
        <label>
          <span>USERNAME</span>
          <input v-model="username" type="text" name="username" placeholder="username…" autocomplete="username" />
        </label>
        <label>
          <span>EMAIL</span>
          <input v-model="email" type="email" name="email" placeholder="user@host…" autocomplete="email" spellcheck="false" />
        </label>
        <label>
          <span>PASSPHRASE</span>
          <input v-model="password" type="password" name="password" placeholder="min 6 chars…" autocomplete="new-password" />
        </label>
        <label>
          <span>CONFIRM PASSPHRASE</span>
          <input v-model="confirm" type="password" name="confirm" placeholder="repeat passphrase…" autocomplete="new-password" />
        </label>

        <div v-if="error" class="error-msg">! {{ error }}</div>

        <button type="submit" class="btn btn-primary" :disabled="loading">
          <span v-if="loading" class="cursor-blink">_</span>
          <span v-else>[+] REGISTER</span>
        </button>
      </form>

      <div class="card-footer">
        <router-link to="/login">[<] BACK TO LOGIN</router-link>
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
