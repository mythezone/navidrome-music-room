// Node 25 exposes its own incomplete localStorage; browser tests must use jsdom's storage.
const browser = (globalThis as unknown as { jsdom: { window: Window } }).jsdom.window
Object.defineProperty(globalThis, 'localStorage', { configurable: true, value: browser.localStorage })
