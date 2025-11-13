<template>
  <div>\
        <!-- TODO add properties -->
        <input v-model="cur.id" type="text" placeholder="ID" />
        <!-- TODO add fields-->
    <button @click="addStation">Add Station</button>
  </div>
</template>

<script lang="ts" setup>
import { main } from '../../wailsjs/go/models'
import { AddSta } from '../../wailsjs/go/main/App'

var cur = new main.Sta()

function addStation() {
    AddSta(cur)
}
</script>