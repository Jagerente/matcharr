import { onScopeDispose, watch } from 'vue'

interface Layer {
  dismiss: () => void
  sticky: boolean
}

interface Entry {
  nav?: number
  seen?: boolean
}

const stack: Layer[] = []

let settled: (fresh: boolean) => void = () => {}
let owned = 0
let pending = 0
let scheduled = false

export function installNav(onSettled: (fresh: boolean) => void): void {
  settled = onSettled
  markSeen()
  window.addEventListener('popstate', onPopState)
}

// Registers a layer for as long as the component lives, or for as long as
// `when` holds. Back dismisses the topmost layer first.
export function useBackLayer(dismiss: () => void, when?: () => boolean): void {
  let layer: Layer | null = null

  const open = () => {
    layer ??= push(dismiss, false)
  }
  const close = () => {
    if (layer) drop(layer)
    layer = null
  }

  if (when) {
    watch(when, (on) => (on ? open() : close()), { immediate: true })
  } else {
    open()
  }
  onScopeDispose(close)
}

// A guard survives its own dismissal: back asks for confirmation instead of
// dropping the user out of a live session.
export function useBackGuard(onBack: () => void): void {
  const layer = push(onBack, true)
  onScopeDispose(() => drop(layer))
}

function push(dismiss: () => void, sticky: boolean): Layer {
  const layer = { dismiss, sticky }
  stack.push(layer)
  schedule()
  return layer
}

function drop(layer: Layer) {
  const i = stack.indexOf(layer)
  if (i === -1) return
  stack.splice(i, 1)
  schedule()
}

// Layers open and close in bursts (a sheet closing while another opens, a view
// swap dropping and pushing a guard), so the depth is settled once per tick.
function schedule() {
  if (scheduled) return
  scheduled = true
  queueMicrotask(reconcile)
}

function reconcile() {
  scheduled = false
  const delta = stack.length - owned
  if (delta === 0) return
  if (delta > 0) {
    while (owned < stack.length) {
      owned++
      history.pushState({ nav: owned, seen: true }, '', location.href)
    }
    return
  }
  owned = stack.length
  pending++
  history.go(delta)
}

function navIndex(state: unknown): number {
  const nav = (state as Entry | null)?.nav
  return typeof nav === 'number' ? nav : 0
}

// Every entry the app lands on is stamped, so an unstamped one is an entry the
// browser has just created: a link pasted over the open tab rather than a step
// back through the stack.
function markSeen() {
  const entry = history.state as Entry | null
  if (entry?.seen) return
  history.replaceState({ ...entry, seen: true }, '', location.href)
}

function onPopState(e: PopStateEvent) {
  const fresh = !(e.state as Entry | null)?.seen
  markSeen()
  const index = navIndex(e.state)
  if (pending > 0) {
    pending--
    owned = index
    schedule()
    settled(fresh)
    return
  }
  // Forward re-enters an entry whose layer is long gone, so undo the move.
  const delta = owned - index
  if (delta < 0) {
    pending++
    history.go(delta)
    return
  }
  owned = index
  dismissTop(delta)
  schedule()
  settled(fresh)
}

// A single gesture can span several entries (a long press on Back), and a
// sticky layer stops the unwinding there.
function dismissTop(count: number) {
  for (let i = 0; i < count; i++) {
    const layer = stack[stack.length - 1]
    if (!layer) return
    if (layer.sticky) {
      layer.dismiss()
      return
    }
    stack.pop()
    layer.dismiss()
  }
}
