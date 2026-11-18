import { readFileSync } from 'node:fs'
import { transformSync } from '@babel/core'
import { describe, expect, it } from 'vitest'
import { messages } from './i18n'

describe('App source regressions', () => {
  it('binds every JSX component, including icons on rarely rendered error screens', () => {
    const source = readFileSync(new URL('./App.jsx', import.meta.url), 'utf8')
    const missing = []
    transformSync(source, {
      configFile: false, babelrc: false, parserOpts: { plugins: ['jsx'] },
      plugins: [() => ({ visitor: {
        JSXOpeningElement(path) {
          let tag = path.node.name
          while (tag.type === 'JSXMemberExpression') tag = tag.object
          if (tag.type === 'JSXIdentifier' && /^[A-Z]/.test(tag.name) && !path.scope.hasBinding(tag.name)) missing.push(tag.name)
        },
        CallExpression(path) {
          if (path.node.callee.name === 't' && path.node.arguments[0]?.type === 'StringLiteral') {
            expect(messages.en[path.node.arguments[0].value], `translation: ${path.node.arguments[0].value}`).toBeTruthy()
          }
        },
      } })],
    })
    expect(missing).toEqual([])
  })

  it('retains the static title required by the Go embedding smoke test', () => {
    expect(readFileSync(new URL('../index.html', import.meta.url), 'utf8')).toContain('<title>听歌房 · Navidrome</title>')
  })
})
