<script>
  export let node
  export let depth = 0
  export let traceStart = 0
  export let traceDuration = 1

  let expanded = depth < 2
  let logsExpanded = false

  $: sp = node.span
  $: logs = node.logs || []
  $: children = node.children || []
  $: hasLogs = logs.length > 0
  $: hasChildren = children.length > 0
  $: canExpand = hasChildren || hasLogs || sp.Attrs?.length

  function pct(ts) {
    return ((new Date(ts).getTime() - traceStart) / traceDuration * 100).toFixed(2)
  }
  function barWidth(start, end) {
    const w = (new Date(end).getTime() - new Date(start).getTime()) / traceDuration * 100
    return Math.max(w, 0.3).toFixed(2)
  }
  function fmtDur(start, end) {
    const ms = new Date(end).getTime() - new Date(start).getTime()
    if (ms < 1) return `<1ms`
    if (ms < 1000) return `${ms.toFixed(1)}ms`
    return `${(ms / 1000).toFixed(2)}s`
  }
  function fmtTime(ts) {
    return new Date(ts).toISOString().replace('T',' ').replace('Z','').slice(11,23)
  }

  const STATUS_COLORS = {
    0:       { bar: '#3b82f6', bg: '#0c2d48' },
    1:       { bar: '#4ade80', bg: '#0b2e1a' },
    2:       { bar: '#ef4444', bg: '#3b1219' },
    'UNSET': { bar: '#3b82f6', bg: '#0c2d48' },
    'OK':    { bar: '#4ade80', bg: '#0b2e1a' },
    'ERROR': { bar: '#ef4444', bg: '#3b1219' },
  }
  $: statusStyle = STATUS_COLORS[sp.Status] || STATUS_COLORS[0]

  const LEVEL_COLORS = {
    TRACE: { bg: '#1a1a2e', fg: '#6b7280' },
    DEBUG: { bg: '#1a1a2e', fg: '#8b949e' },
    INFO:  { bg: '#0c2d48', fg: '#58a6ff' },
    WARN:  { bg: '#2d2305', fg: '#f0b429' },
    ERROR: { bg: '#3b1219', fg: '#f87171' },
    FATAL: { bg: '#450a0a', fg: '#fca5a5' },
  }

  const INDENT = 20
</script>

<div class="span-wrap">
  <div class="span-row" role="button" tabindex="0"
    on:click={() => { if (canExpand) expanded = !expanded }}
    on:keydown={e => { if ((e.key === 'Enter' || e.key === ' ') && canExpand) { e.preventDefault(); expanded = !expanded } }}>
    <!-- Left: name column -->
    <div class="name-col" style="padding-left:{depth * INDENT + 8}px">
      {#if canExpand}
        <span class="toggle" class:open={expanded}>
          <svg viewBox="0 0 8 8" fill="none"><path d="M2 3l2 2 2-2" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"/></svg>
        </span>
      {:else}
        <span class="toggle dot">
          <svg viewBox="0 0 8 8" fill="none"><circle cx="4" cy="4" r="1.5" fill="currentColor"/></svg>
        </span>
      {/if}

      <span class="service">{sp.Service}</span>
      <span class="op">{sp.Operation}</span>

      {#if hasLogs}
        <span class="log-count">{logs.length}</span>
      {/if}
      {#if sp.Status === 2 || sp.Status === 'ERROR'}
        <span class="error-dot"></span>
      {/if}
    </div>

    <!-- Right: timeline bar -->
    <div class="bar-col">
      <div class="bar-track">
        <div class="bar" style="left:{pct(sp.StartTime)}%;width:{barWidth(sp.StartTime,sp.EndTime)}%;background:{statusStyle.bar}"></div>
      </div>
      <span class="dur">{fmtDur(sp.StartTime, sp.EndTime)}</span>
    </div>
  </div>

  {#if expanded}
    <!-- Span attributes -->
    <div class="detail" style="padding-left:{depth * INDENT + 36}px">
      <div class="attr-grid">
        <span class="attr-key">span_id</span>
        <span class="attr-val mono">{sp.SpanID}</span>

        <span class="attr-key">start</span>
        <span class="attr-val mono">{fmtTime(sp.StartTime)}</span>

        {#if sp.Status !== undefined}
          <span class="attr-key">status</span>
          <span class="attr-val">
            <span class="status-pill" style="background:{statusStyle.bg};color:{statusStyle.bar}">
              {sp.Status === 2 || sp.Status === 'ERROR' ? 'ERROR' : sp.Status === 1 || sp.Status === 'OK' ? 'OK' : 'UNSET'}
            </span>
          </span>
        {/if}

        {#each sp.Attrs || [] as a}
          <span class="attr-key">{a.Key}</span>
          <span class="attr-val">{a.Value}</span>
        {/each}
      </div>
    </div>

    <!-- Logs attached to this span -->
    {#if hasLogs}
      <div class="logs-section" style="padding-left:{depth * INDENT + 36}px">
        <button class="logs-toggle" on:click|stopPropagation={() => logsExpanded = !logsExpanded}>
          <svg class="logs-chevron" class:open={logsExpanded} viewBox="0 0 8 8" fill="none"><path d="M2 3l2 2 2-2" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"/></svg>
          Logs
          <span class="logs-count">{logs.length}</span>
        </button>
        {#if logsExpanded}
          <div class="logs-list">
            {#each logs as log}
              <div class="log-entry">
                <span class="log-time">{fmtTime(log.Timestamp)}</span>
                <span class="log-lvl" style="background:{(LEVEL_COLORS[log.Level] || LEVEL_COLORS.DEBUG).bg};color:{(LEVEL_COLORS[log.Level] || LEVEL_COLORS.DEBUG).fg}">{log.Level}</span>
                <span class="log-body">{log.Body}</span>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}

    <!-- Child spans -->
    {#each children as child}
      <svelte:self node={child} depth={depth + 1} {traceStart} {traceDuration} />
    {/each}
  {/if}
</div>

<style>
  .span-wrap { border-bottom: 1px solid #111820; }

  .span-row {
    display: flex; align-items: center; cursor: pointer; height: 34px;
    min-width: 0; transition: background 0.1s;
  }
  .span-row:hover { background: #131921; }

  .name-col {
    display: flex; align-items: center; gap: 6px;
    width: 320px; flex-shrink: 0; overflow: hidden; white-space: nowrap;
  }

  .toggle {
    display: flex; align-items: center; justify-content: center;
    width: 16px; height: 16px; flex-shrink: 0;
    color: #4a5568; transition: transform 0.15s;
  }
  .toggle svg { width: 10px; height: 10px; }
  .toggle:not(.open) { transform: rotate(-90deg); }
  .toggle.dot { transform: none; }

  .service { color: #6b7994; font-size: 11px; flex-shrink: 0; }
  .op {
    color: #c4b5fd; font-size: 12px; font-weight: 500;
    overflow: hidden; text-overflow: ellipsis;
  }

  .log-count {
    background: #0c2d48; color: #58a6ff; border-radius: 10px;
    font-size: 9px; font-weight: 600; padding: 1px 6px; flex-shrink: 0;
  }
  .error-dot {
    width: 6px; height: 6px; border-radius: 50%;
    background: #ef4444; flex-shrink: 0;
  }

  .bar-col { flex: 1; display: flex; align-items: center; gap: 8px; padding: 0 12px; min-width: 0; }
  .bar-track {
    flex: 1; height: 10px; position: relative;
    background: #111820; border-radius: 3px; overflow: hidden;
  }
  .bar {
    position: absolute; top: 0; height: 100%; border-radius: 3px;
    opacity: 0.8; min-width: 2px; transition: width 0.2s ease;
  }
  .dur {
    font-size: 11px; width: 60px; text-align: right; flex-shrink: 0;
    font-family: 'SF Mono', 'Fira Code', monospace;
    color: #6b7994;
  }

  /* --- Detail --- */
  .detail {
    padding: 6px 12px 8px; background: #0b0f14;
    border-bottom: 1px solid #111820;
  }
  .attr-grid {
    display: grid; grid-template-columns: auto 1fr;
    gap: 2px 16px; font-size: 12px;
  }
  .attr-key {
    color: #4a5568;
    font-family: 'SF Mono', 'Fira Code', monospace; font-size: 11px;
  }
  .attr-val { color: #c9d1d9; word-break: break-all; }
  .mono { font-family: 'SF Mono', 'Fira Code', monospace; font-size: 11px; }

  .status-pill {
    display: inline-block; font-size: 10px; font-weight: 700;
    padding: 1px 8px; border-radius: 4px;
    letter-spacing: 0.03em;
  }

  /* --- Logs --- */
  .logs-section {
    padding: 4px 12px 8px; background: #0b0f14;
  }
  .logs-toggle {
    display: flex; align-items: center; gap: 6px;
    background: none; border: none; color: #58a6ff;
    cursor: pointer; padding: 4px 0;
    font-size: 11px; font-weight: 600; font-family: inherit;
  }
  .logs-toggle:hover { color: #79b8ff; }
  .logs-chevron {
    width: 10px; height: 10px; color: #4a5568;
    transition: transform 0.15s;
  }
  .logs-chevron:not(.open) { transform: rotate(-90deg); }
  .logs-count {
    background: #0c2d48; color: #58a6ff; border-radius: 10px;
    font-size: 9px; padding: 0 5px; font-weight: 600;
  }

  .logs-list { padding-top: 4px; }
  .log-entry {
    display: flex; gap: 8px; align-items: baseline;
    padding: 3px 0; font-size: 12px;
  }
  .log-time {
    width: 90px; flex-shrink: 0;
    font-family: 'SF Mono', 'Fira Code', monospace;
    font-size: 10px; color: #4a5568;
  }
  .log-lvl {
    font-size: 9px; font-weight: 700; padding: 1px 6px; border-radius: 3px;
    flex-shrink: 0; min-width: 38px; text-align: center;
  }
  .log-body { color: #c9d1d9; word-break: break-all; line-height: 1.4; }
</style>
