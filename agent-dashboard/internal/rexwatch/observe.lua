-- Ephemeral, read-only observer run LOCALLY by the official Rex CLI. This is
-- embedded in the native dashboard; it is not installed on the observed host.
local options = rex.args
local terminal = "com.superlogical.terminal"
local identity
local sequence = 0
local panes = {}
local attached_sessions = {}
local selected = {}
for _, id in ipairs(options.sessions or {}) do selected[id] = true end
local restricted = next(selected) ~= nil

local function call(method, args)
  local result, err = rex.call(method, args)
  if result == nil then error(err or ("no result from " .. method), 0) end
  return result
end

local function verify()
  local current = call("server.status")
  if current.host_id ~= options.host_id then
    error("Rex host identity changed; refusing to observe this endpoint", 0)
  end
  if identity and current.instance_id ~= identity.instance_id then
    error("Rex server instance changed; a fresh snapshot is required", 0)
  end
  identity = current
end

local function emit(kind, fields)
  fields = fields or {}
  sequence = sequence + 1
  fields.agent_dashboard_rex = 1
  fields.type = kind
  fields.seq = sequence
  fields.host_id = identity.host_id
  fields.instance_id = identity.instance_id
  assert(io.stdout:write(assert(rex.json.encode(fields)), "\n"))
  assert(io.stdout:flush())
end

local function key(session_id, block_id)
  return assert(rex.json.encode({session_id, block_id}))
end

local function reports(p)
  local result = call(terminal .. ".program_status", {
    session_id = p.session_id, block_id = p.block_id, args = {},
  })
  local records = {}
  for _, r in ipairs(result.records or {}) do
    if #records >= 4096 then error("Rex program record limit exceeded", 0) end
    -- Deliberately exclude process owners, paths, arguments and other metadata.
    records[#records + 1] = {
      id = r.id or "",
      app = r.app, effective_app = r.effective_app, title = r.title,
      state = r.state, kind = r.kind, msg = r.msg,
      progress = r.progress, updated_at = r.updated_at,
    }
  end
  p.records = records
end

local function snapshot()
  verify()
  local result, next_panes, next_attached, count, records, excluded, sessions = {}, {}, {}, 0, 0, 0, 0
  for _, s in ipairs(call("session.list").sessions or {}) do
    if not restricted or selected[s.session_id] then
      sessions = sessions + 1
      if sessions > 64 then error("Rex session limit exceeded", 0) end
      if not attached_sessions[s.session_id] then
        local attached, err = rex.session.attach{session_id = s.session_id}
        if attached == nil then error(err, 0) end
        attached_sessions[s.session_id] = true
      end
      next_attached[s.session_id] = true
      emit("heartbeat") -- API progress during a possibly slow staged baseline
      for _, b in ipairs(call("session.list_blocks", {session_id = s.session_id}).blocks or {}) do
        if b.creator_name == terminal then
          if b.host_id ~= identity.host_id then
            excluded = excluded + 1
          else
            count = count + 1
            if count > 256 then error("Rex pane limit exceeded", 0) end
            local p = {session_id = s.session_id, session_label = s.label or "",
              block_id = b.block_id, block_label = b.label or ""}
            reports(p)
            emit("heartbeat")
            records = records + #p.records
            if records > 4096 then error("Rex total program record limit exceeded", 0) end
            next_panes[key(p.session_id, p.block_id)] = p
            result[#result + 1] = p
          end
        end
      end
    end
  end
  verify()
  panes = next_panes
  attached_sessions = next_attached
  emit("snapshot", {panes = result, excluded_panes = excluded})
end

local function update(ctx)
  if restricted and not selected[ctx.session_id] then return end
  local p = panes[key(ctx.session_id, ctx.block_id)]
  if p == nil then snapshot(); return end
  -- Events invalidate a pane. Re-read its CURRENT full report list instead of
  -- replaying an event older than the first snapshot. Lua dispatch is serial.
  -- Changed AND removed events therefore cannot resurrect an old report.
  reports(p)
  verify()
  emit("pane", {pane = p})
end

-- Register before the first snapshot. These are instance-local subscriptions,
-- not Rex configuration hooks. --fail-fast turns any handler error into a
-- reconnect followed by a new full baseline in the native owner process.
rex.on("terminal.program_status_changed", update)
rex.on("terminal.program_status_removed", update)
rex.on("session_view_changed", function(ctx)
  if not restricted or selected[ctx.session_id] then snapshot() end
end)
rex.on("session_created", function() if not restricted then snapshot() end end)
rex.on("session_destroyed", function(ctx, ev)
  local id = ctx.session_id or ev.session_id
  if restricted and id and not selected[id] then return end
  if id then attached_sessions[id] = nil end
  snapshot()
end)

verify()
emit("hello")
snapshot()
rex.every(10, function() verify(); emit("heartbeat") end)
-- A bounded reconciliation covers undocumented CLI subscription-startup gaps
-- and structural events lost by a server/CLI version. It is not the live path.
rex.every(60, snapshot)
