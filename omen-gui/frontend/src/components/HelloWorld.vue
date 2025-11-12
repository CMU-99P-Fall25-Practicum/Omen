<script lang="ts" setup>
import {reactive} from 'vue'
import {GenerateJSON, Greet} from '../../wailsjs/go/main/App'
import { main } from '../../wailsjs/go/models'


//#region local data

// data that must update the UI automatically when changed/set
const dynamic = reactive({
  name: "",
  resultText: "Please enter your name below 👇",
  gend: "",
  dropdown1: "",
  dropdown2: "",
  dropdown3: "",
})

// the complete set of values to pass to the json generator
var input: main.Input = new main.Input({
    schema: "0.0",
    topo: new main.Topo({
      nets: new main.Nets({
        noise_th: -100,
        propagation_model: new main.PropagationModel({model: "",exp: 0,s: 0})
      })
    })
  })

// options for all dropdown menus
const dropdownOptions = {
  dropdown1: [
    { label: 'Option 1', value: 'option1' },
    { label: 'Option 2', value: 'option2' },
    { label: 'Option 3', value: 'option3' },
  ],
  dropdown2: [
    { label: 'Option A', value: 'optionA' },
    { label: 'Option B', value: 'optionB' },
    { label: 'Option C', value: 'optionC' },
  ],
  dropdown3: [
    { label: 'Option X', value: 'optionX' },
    { label: 'Option Y', value: 'optionY' },
    { label: 'Option Z', value: 'optionZ' },
  ],
}
//#endregion local data

function greet() {
  Greet(dynamic.name).then(result => {
    dynamic.resultText = result
  })
}

function generateJSON(){
    GenerateJSON(input).then(success => {
      if (success){dynamic.gend = "successfully generated input file"}
      else {dynamic.gend = "an error occurred"}
    })
    

}

</script>

<template>
  <main>
    <div class="container">
      <div id="input" class="input-box">
        <input id="name" v-model="dynamic.name" autocomplete="off" class="input" type="text"/>
        <button class="btn" @click="greet">Greet</button>
      </div>
      <div class="dropdowns">
        <select v-model="dynamic.dropdown1">
          <option v-for="option in dropdownOptions.dropdown1" :key="option.value" :value="option.value">
            {{ option.label }}
          </option>
        </select>
        <select v-model="dynamic.dropdown2">
          <option v-for="option in dropdownOptions.dropdown2" :key="option.value" :value="option.value">
            {{ option.label }}
          </option>
        </select>
        <select v-model="dynamic.dropdown3">
          <option v-for="option in dropdownOptions.dropdown3" :key="option.value" :value="option.value">
            {{ option.label }}
          </option>
        </select>
      </div>
    </div>
    <div id="result" class="result">{{ dynamic.resultText }}</div>
    <div id="gen-confirm" class="result">{{ dynamic.gend }}</div>
    <div id="generate"> 
      <button class="btn" @click="generateJSON">Generate</button>
    </div>
  </main>
</template>


<style scoped>
.result {
  height: 20px;
  line-height: 20px;
  margin: 1.5rem auto;
}

.input-box .btn {
  width: 60px;
  height: 30px;
  line-height: 30px;
  border-radius: 3px;
  border: none;
  margin: 0 0 0 20px;
  padding: 0 8px;
  cursor: pointer;
}


.input-box .btn:hover {
  background-image: linear-gradient(to top, #cfd9df 0%, #e2ebf0 100%);
  color: #333333;
}

.input-box .input {
  border: none;
  border-radius: 3px;
  outline: none;
  height: 30px;
  line-height: 30px;
  padding: 0 10px;
  background-color: rgba(240, 240, 240, 1);
  -webkit-font-smoothing: antialiased;
}

.input-box .input:hover {
  border: none;
  background-color: rgba(255, 255, 255, 1);
}

.input-box .input:focus {
  border: none;
  background-color: rgba(255, 255, 255, 1);
}

.container {
  display: flex;
}

.dropdowns {
  margin-left: 20px;
}

.dropdowns select {
  width: 150px;
  height: 30px;
  line-height: 30px;
  border-radius: 3px;
  border: none;
  margin-bottom: 10px;
}

.dropdowns select:hover {
  background-color: rgba(255, 255, 255, 1);
}

</style>
