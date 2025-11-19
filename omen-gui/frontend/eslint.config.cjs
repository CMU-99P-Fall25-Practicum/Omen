// eslint.config.cjs
const vuePlugin = require('eslint-plugin-vue')
const vueParser = require('vue-eslint-parser')
const tsParser = require('@typescript-eslint/parser')
const tsPlugin = require('@typescript-eslint/eslint-plugin')

module.exports = [
  {
    files: ['**/*.{js,cjs,mjs,ts,tsx,vue}'],
    ignores: ['node_modules/**', 'dist/**', 'coverage/**'],

    languageOptions: {
      parser: vueParser,
      parserOptions: {
        parser: tsParser,
        extraFileExtensions: ['.vue'],
        ecmaVersion: 2024,
        sourceType: 'module',
        ecmaFeatures: { jsx: true }
      },
    },

    plugins: {
      vue: vuePlugin,
      '@typescript-eslint': tsPlugin
    },

    rules: {
      /* Vue formatting/structure rules */
      'vue/html-closing-bracket-newline': ['error', { singleline: 'never', multiline: 'never' }],
      'vue/html-closing-bracket-spacing': ['error', { startTag: 'never', endTag: 'never', selfClosingTag: 'always' }],
      'vue/html-self-closing': [
        'error',
        {
          html: { void: 'never', normal: 'always', component: 'always' },
          svg: 'always',
          math: 'always'
        }
      ],
      'vue/max-attributes-per-line': ['error', { singleline: 3, multiline: { max: 1 } }],

      /* TypeScript rules */
      '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_', varsIgnorePattern: '^_' }],
      '@typescript-eslint/explicit-module-boundary-types': 'off',
      '@typescript-eslint/no-explicit-any': 'warn',

      /* Stylistic JS rules (ESLint will autofix where possible) */
      'semi': ['error', 'never'],
      'quotes': ['error', 'single', { avoidEscape: true }],

      'no-console': ['warn', { allow: ['warn', 'error'] }],
      'no-debugger': 'warn'
    }
  },

  {
    files: ['**/*.test.*', '**/__tests__/**'],
    rules: {
      'no-console': 'off',
      '@typescript-eslint/no-explicit-any': 'off'
    }
  }
]
