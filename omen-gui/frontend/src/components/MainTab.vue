<script lang="ts" setup>
  import { isValid as IsValidIP } from 'ipaddr.js'
  import { ref, computed, watchEffect } from 'vue'

  const emit = defineEmits<{
    valid: [is: boolean] // is currently valid?
  }>()

  let username = ref(''),
    password = ref(''),
    host = ref('127.0.0.1'),
    port = ref(22)

  // check each field for validation errors whenever one changes
  let validationErrors = computed(() => {
    const msgs: string[] = []

    if (username.value.trim() === '') msgs.push('SSH username cannot be empty')
    if (host.value.trim() === '') msgs.push('SSH host cannot be empty')
    else {
      // populated-ony checks
      if (!IsValidIP(host.value)) msgs.push('SSH host must be a valid IPv4 or IPv6 address')
    }
    if (port.value < 1 || port.value > (2 << 16) - 1) msgs.push('Port must be between 1 and 65535')

    return msgs
  })

  watchEffect(() => {
    emit('valid', validationErrors.value.length === 0)
  })
</script>

<template>
  <div>
    <label>Username</label>: <input v-model="username" type="text"> <label>Password</label>:
    <input v-model="password" type="password">
  </div>
  <div>
    <label>Host</label>: <input v-model="host" type="text"> <label>Port</label>:
    <input v-model="port"
type="number"
min="1"
max="65535">
  </div>
  <div class="error-list">
    <div v-for="(err, idx) in validationErrors" :key="idx">{{ err }}</div>
  </div>
</template>
