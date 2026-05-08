/**
 * Global Vitest setup. Resets the Pinia store registration before
 * each test so state from one spec never leaks to the next, and
 * stubs localStorage so the persisted-state plugin doesn't throw on
 * boot in the happy-dom env.
 */
import { afterEach, beforeEach } from 'vitest'

beforeEach(() => {
  // happy-dom ships localStorage but reset between tests for sanity.
  window.localStorage.clear()
  // dataset clean-up so theme assertions start from a known state.
  document.documentElement.removeAttribute('data-preset')
  document.documentElement.lang = 'en'
})

afterEach(() => {
  window.localStorage.clear()
})
