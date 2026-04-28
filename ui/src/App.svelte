<script>
  import { onMount } from 'svelte'
  import Logs from './lib/Logs.svelte'
  import Traces from './lib/Traces.svelte'
  import TraceView from './lib/TraceView.svelte'

  let route = location.hash || '#/logs'
  let traceId = ''

  function parseRoute(hash) {
    const m = hash.match(/^#\/traces\/([a-f0-9]{32})$/)
    if (m) { traceId = m[1]; return '#/trace' }
    traceId = ''
    return hash || '#/logs'
  }

  $: currentRoute = parseRoute(route)

  onMount(() => {
    const handler = () => { route = location.hash }
    window.addEventListener('hashchange', handler)
    return () => window.removeEventListener('hashchange', handler)
  })
</script>

<div class="app">
  <nav>
    <a class="logo" href="#/logs">
      <img class="logo-icon" src="/logo.png" alt="Amber">
      amber
    </a>

    <div class="nav-links">
      <a class:active={currentRoute === '#/logs'} href="#/logs">
        <svg viewBox="0 0 16 16" fill="none"><path d="M2 4h12M2 8h10M2 12h8" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/></svg>
        Logs
      </a>
      <a class:active={currentRoute === '#/traces' || currentRoute === '#/trace'} href="#/traces">
        <svg viewBox="0 0 16 16" fill="none"><path d="M2 4h4M6 8h6M4 12h8" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/><circle cx="3" cy="8" r="1.2" fill="currentColor"/></svg>
        Traces
      </a>
    </div>
  </nav>

  <main>
    {#if currentRoute === '#/logs'}
      <Logs />
    {:else if currentRoute === '#/traces'}
      <Traces />
    {:else if currentRoute === '#/trace'}
      <TraceView id={traceId} />
    {/if}
  </main>
</div>

<style>
  :global(*, *::before, *::after) { box-sizing: border-box; margin: 0; padding: 0; }
  :global(body) {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Inter', sans-serif;
    background: #0a0e14;
    color: #e2e8f0;
    font-size: 13px;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
  }
  :global(a) { color: inherit; }
  :global(::-webkit-scrollbar) { width: 6px; height: 6px; }
  :global(::-webkit-scrollbar-track) { background: transparent; }
  :global(::-webkit-scrollbar-thumb) { background: #30363d; border-radius: 3px; }
  :global(::-webkit-scrollbar-thumb:hover) { background: #484f58; }

  .app { display: flex; flex-direction: column; height: 100vh; }

  nav {
    display: flex; align-items: center; gap: 0;
    padding: 0 20px; height: 48px;
    background: #0d1117;
    border-bottom: 1px solid #1b2230;
    flex-shrink: 0;
  }

  .logo {
    display: flex; align-items: center; gap: 8px;
    font-weight: 700; color: #f59e0b; text-decoration: none;
    font-size: 15px; letter-spacing: -0.3px;
    margin-right: 24px;
    padding: 6px 0;
  }
  .logo-icon { width: 20px; height: 20px; color: #f59e0b; }

  .nav-links {
    display: flex; align-items: center; gap: 2px;
    height: 100%;
  }
  .nav-links a {
    display: flex; align-items: center; gap: 6px;
    color: #6b7994; text-decoration: none;
    padding: 6px 14px; border-radius: 8px;
    font-size: 13px; font-weight: 500;
    transition: all 0.15s ease;
    position: relative;
  }
  .nav-links a svg { width: 14px; height: 14px; flex-shrink: 0; }
  .nav-links a:hover { color: #c9d1d9; background: rgba(255,255,255,0.04); }
  .nav-links a.active {
    color: #e2e8f0;
    background: rgba(255,255,255,0.06);
  }

  main { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
</style>
