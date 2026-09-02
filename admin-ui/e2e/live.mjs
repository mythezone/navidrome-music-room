import { chromium } from 'playwright-core'
import path from 'node:path'

const baseURL = (process.env.NMR_TEST_BASE_URL || '').replace(/\/$/, '')
const username = process.env.NMR_TEST_USERNAME || ''
const password = process.env.NMR_TEST_PASSWORD || ''
const screenshotDir = process.env.NMR_SCREENSHOT_DIR || path.resolve('../docs/assets')
const executablePath = process.env.CHROMIUM_PATH || '/usr/bin/chromium-browser'

if (!baseURL || !username || !password) {
  throw new Error('Set NMR_TEST_BASE_URL, NMR_TEST_USERNAME, and NMR_TEST_PASSWORD')
}

const browser = await chromium.launch({ executablePath, headless: true, args: ['--no-sandbox'] })
const context = await browser.newContext({ viewport: { width: 1440, height: 1050 }, colorScheme: 'dark' })
const page = await context.newPage()
const browserErrors = []
page.on('pageerror', (error) => browserErrors.push(error.message))
page.on('console', (message) => {
  if (message.type() === 'error') browserErrors.push(message.text())
})

try {
  const loginResponse = await context.request.post(`${baseURL}/auth/login`, { data: { username, password } })
  if (!loginResponse.ok()) throw new Error(`Navidrome login failed: ${loginResponse.status()}`)
  const auth = await loginResponse.json()

  await page.goto(`${baseURL}/app/`)
  await page.evaluate((value) => {
    localStorage.setItem('token', value.token)
    localStorage.setItem('userId', value.id)
    localStorage.setItem('name', value.name)
    localStorage.setItem('username', value.username)
    localStorage.setItem('role', value.isAdmin ? 'admin' : 'regular')
    localStorage.setItem('subsonic-salt', value.subsonicSalt)
    localStorage.setItem('subsonic-token', value.subsonicToken)
    localStorage.setItem('is-authenticated', 'true')
  }, auth)

  await page.goto(`${baseURL}/app/#/plugin/navidrome-music-room/show`)
  const websiteLink = page.locator('a[href="/music-room/admin/"]')
  await websiteLink.waitFor({ state: 'visible', timeout: 15000 })
  await page.screenshot({ path: path.join(screenshotDir, 'navidrome-plugin-entry.png'), fullPage: true })

  // Navidrome's own UI logs Cache API errors on non-secure HTTP IP origins.
  // Only fail this smoke test for errors emitted by the Music Room page itself.
  browserErrors.length = 0
  await page.goto(`${baseURL}/music-room/admin/`)
  await page.getByText('房间服务已连接').waitFor({ state: 'visible', timeout: 15000 })
  const managedRoomName = 'Navidrome 管理页测试房'
  if (await page.getByText(managedRoomName).count() === 0) {
    await page.getByRole('button', { name: '创建' }).click()
    const roomDialog = page.locator('div[role="dialog"]')
    await roomDialog.locator('#room-name').fill(managedRoomName)
    const roomResponsePromise = page.waitForResponse((response) => response.url().endsWith('/api/v1/rooms') && response.request().method() === 'POST')
    await roomDialog.getByRole('button', { name: '创建房间' }).click()
    const roomResponse = await roomResponsePromise
    if (!roomResponse.ok()) throw new Error(`Room creation failed: ${roomResponse.status()}`)
  }
  await page.getByText(managedRoomName).first().click()
  await page.getByRole('heading', { name: managedRoomName }).waitFor({ state: 'visible' })
  await page.locator('.MuiSnackbar-root').waitFor({ state: 'hidden', timeout: 5000 }).catch(() => {})
  await page.screenshot({ path: path.join(screenshotDir, 'admin-ui-live.png'), fullPage: true })

  await page.getByRole('button', { name: '分享链接 / 二维码' }).click()
  const shareDialog = page.locator('div[role="dialog"]')
  await shareDialog.waitFor({ state: 'visible' })
  await shareDialog.locator('#invite-label').fill('管理页联调邀请')
  const inviteResponsePromise = page.waitForResponse((response) => response.url().includes('/api/v1/rooms/') && response.url().endsWith('/invites') && response.request().method() === 'POST')
  await page.getByRole('button', { name: '生成链接和二维码' }).click()
  const inviteResponse = await inviteResponsePromise
  if (!inviteResponse.ok()) throw new Error(`Invitation creation failed: ${inviteResponse.status()}`)
  const createdInvite = await inviteResponse.json()
  await page.getByText('邀请已生成').waitFor({ state: 'visible', timeout: 10000 })
  const shareURL = createdInvite.shareURL
  if (!shareURL?.includes('/music-room/join/') || !shareURL.includes('#invite=')) {
    throw new Error(`Generated share URL is malformed: ${shareURL}`)
  }
  await page.locator('div[role="dialog"]').screenshot({ path: path.join(screenshotDir, 'share-dialog-live.png') })
  await page.getByRole('button', { name: '完成' }).click()
  await shareDialog.waitFor({ state: 'hidden' })

  // Keep the screenshot useful without leaving a live invitation secret in the
  // repository. The dialog screenshot still shows the tested URL, but the
  // matching server-side invitation is revoked before the test succeeds.
  const inviteRow = page.getByRole('listitem').filter({ hasText: '管理页联调邀请' }).first()
  const revokeResponsePromise = page.waitForResponse((response) => response.url().includes(`/invites/${createdInvite.inviteID}`) && response.request().method() === 'DELETE')
  await inviteRow.getByRole('button', { name: '撤销' }).click()
  const revokeResponse = await revokeResponsePromise
  if (!revokeResponse.ok()) throw new Error(`Invitation revocation failed: ${revokeResponse.status()}`)
  await page.locator('.MuiSnackbar-root').waitFor({ state: 'hidden', timeout: 5000 }).catch(() => {})

  await page.setViewportSize({ width: 390, height: 844 })
  await page.screenshot({ path: path.join(screenshotDir, 'admin-ui-mobile.png'), fullPage: true })
  process.stdout.write(`${shareURL}\n`)

  if (browserErrors.length) throw new Error(`Browser errors:\n${browserErrors.join('\n')}`)
} finally {
  await browser.close()
}
