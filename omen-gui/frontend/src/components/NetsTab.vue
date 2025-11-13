<template>
  <div>
    <input v-model="noise_threshold" type="number" placeholder="Noise Threshold" />
    <select v-model="model">
      <option value=Friis>Friis</option>
      <option value=LogDistance>Log-Distance</option>
    </select>
    <input v-model="exp" type="number" placeholder="Exponent" />
    <input v-model="s" type="number" placeholder="S" />
  </div>
</template>

<script lang="ts" setup>
import { main } from '../../wailsjs/go/models'

var noise_threshold: number
var model: main.PropModel
var exp: number
var s: number

</script>
