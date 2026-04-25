const form = document.getElementById("filters")
const limitInput = document.getElementById("limit")
const searchInput = document.getElementById("job-search")
const scheduleSelect = document.getElementById("schedule-select")
const startDateInput = document.getElementById("start-date")
const endDateInput = document.getElementById("end-date")
const resetButton = document.getElementById("reset-button")
const statusNode = document.getElementById("status")
const summaryNode = document.getElementById("summary")
const eventsNode = document.getElementById("events")
const statCountNode = document.getElementById("stat-count")
const statLatestNode = document.getElementById("stat-latest")
const eventTemplate = document.getElementById("event-template")

const state = {
  audits: [],
  schedules: new Map(),
  request: null,
}

const timestampFormat = new Intl.DateTimeFormat(undefined, {
  dateStyle: "medium",
  timeStyle: "short",
})

document.addEventListener("DOMContentLoaded", init)

async function init() {
  bindEvents()
  renderLoading()
  await Promise.allSettled([loadSchedules(), loadAudits()])
}

function bindEvents() {
  form.addEventListener("submit", (event) => {
    event.preventDefault()
    loadAudits()
  })

  scheduleSelect.addEventListener("change", () => {
    loadAudits()
  })

  limitInput.addEventListener("change", () => {
    loadAudits()
  })

  startDateInput.addEventListener("change", () => {
    normalizeDateRange()
    loadAudits()
  })

  endDateInput.addEventListener("change", () => {
    normalizeDateRange()
    loadAudits()
  })

  searchInput.addEventListener("input", () => {
    renderAudits()
  })

  resetButton.addEventListener("click", () => {
    form.reset()
    limitInput.value = "25"
    renderLoading()
    loadAudits()
  })
}

async function loadSchedules() {
  try {
    const schedules = await fetchJSON("/schedules")
    state.schedules = new Map(
      schedules.map((schedule) => [String(schedule.id), normalizeSchedule(schedule)])
    )
    updateScheduleOptions()
  } catch (error) {
    setStatus("Loaded audits without schedule metadata. The dashboard can still search by script path.", "error")
  }
}

async function loadAudits() {
  const filters = readFilters()
  const request = buildRequest(filters)

  if (state.request) {
    state.request.abort()
  }

  const controller = new AbortController()
  state.request = controller

  renderLoading()
  setStatus("Loading audit events...", "")

  try {
    const audits = await fetchJSON(request.path, controller.signal)
    if (state.request !== controller) {
      return
    }

    state.audits = applyClientFilters(audits.map(normalizeAudit), filters)
    updateScheduleOptions()
    renderAudits()
    setStatus(buildStatusMessage(filters, state.audits.length), "success")
  } catch (error) {
    if (error.name === "AbortError") {
      return
    }
    state.audits = []
    renderAudits()
    setStatus(error.message || "Failed to load audit events.", "error")
  } finally {
    if (state.request === controller) {
      state.request = null
    }
  }
}

function renderLoading() {
  summaryNode.textContent = "Loading audit events..."
  statCountNode.textContent = "0"
  statLatestNode.textContent = "-"
  eventsNode.innerHTML = '<div class="loading-state">Fetching the latest scheduled job activity.</div>'
}

function renderAudits() {
  const filters = readFilters()
  const visibleAudits = filterBySearch(state.audits, filters.jobSearch)

  renderSummary(visibleAudits, filters)
  statCountNode.textContent = String(visibleAudits.length)
  statLatestNode.textContent = visibleAudits[0] ? formatRelativeDate(visibleAudits[0].startTime) : "-"

  if (visibleAudits.length === 0) {
    eventsNode.innerHTML = '<div class="empty-state">No audit events match the current filters.</div>'
    return
  }

  const fragment = document.createDocumentFragment()
  for (const audit of visibleAudits) {
    fragment.appendChild(renderAuditCard(audit))
  }

  eventsNode.replaceChildren(fragment)
}

function renderSummary(audits, filters) {
  const scope = filters.scheduleId
    ? `for ${scheduleLabelById(filters.scheduleId)}`
    : "across all jobs"
  const dateRange = formatDateRange(filters.startDate, filters.endDate)
  const search = filters.jobSearch ? ` matching "${filters.jobSearch}"` : ""
  summaryNode.textContent =
    `Showing ${audits.length} most recent event` +
    `${audits.length === 1 ? "" : "s"} ${scope}${dateRange}${search}.`
}

function renderAuditCard(audit) {
  const node = eventTemplate.content.firstElementChild.cloneNode(true)
  const status = normalizeStatus(audit.status)
  const detailsNode = node.querySelector(".event-details")
  const errorBlock = node.querySelector(".detail-error-block")
  const outputBlock = node.querySelector(".detail-output-block")
  const previewNode = node.querySelector(".event-preview")

  node.querySelector(".event-job").textContent = audit.displayName
  node.querySelector(".event-script").textContent = audit.scriptPath || "No script path recorded"
  node.querySelector(".event-schedule").textContent = `#${audit.scheduleId || "unknown"}`
  node.querySelector(".event-start").textContent = formatTimestamp(audit.startTime)
  node.querySelector(".event-duration").textContent = formatDuration(audit.startTime, audit.endTime)

  const statusNode = node.querySelector(".event-status")
  statusNode.textContent = status
  statusNode.dataset.status = status

  const preview = audit.error || audit.output || "No output recorded."
  previewNode.textContent = truncate(preview, 220)
  previewNode.classList.toggle("is-error", status === "failure")

  if (audit.error) {
    node.querySelector(".event-error").textContent = audit.error
  } else {
    errorBlock.hidden = true
  }

  if (audit.output) {
    node.querySelector(".event-output").textContent = audit.output
  } else {
    outputBlock.hidden = true
  }

  if (!audit.error && !audit.output) {
    detailsNode.hidden = true
  }

  return node
}

function updateScheduleOptions() {
  const selected = scheduleSelect.value
  const options = collectScheduleOptions()

  if (selected && !options.some((option) => option.id === selected)) {
    options.unshift({
      id: selected,
      label: scheduleLabelById(selected),
    })
  }

  scheduleSelect.innerHTML = ""
  scheduleSelect.appendChild(new Option("All jobs", ""))

  for (const option of options) {
    scheduleSelect.appendChild(new Option(option.label, option.id))
  }

  if (options.some((option) => option.id === selected)) {
    scheduleSelect.value = selected
  }
}

function collectScheduleOptions() {
  const options = new Map()

  for (const schedule of state.schedules.values()) {
    options.set(schedule.id, {
      id: schedule.id,
      label: schedule.label,
    })
  }

  for (const audit of state.audits) {
    if (!audit.scheduleId || options.has(audit.scheduleId)) {
      continue
    }
    options.set(audit.scheduleId, {
      id: audit.scheduleId,
      label: `${audit.displayName} (#${audit.scheduleId})`,
    })
  }

  return Array.from(options.values()).sort((left, right) =>
    left.label.localeCompare(right.label)
  )
}

function readFilters() {
  normalizeDateRange()
  const limit = clampLimit(limitInput.value)
  limitInput.value = String(limit)

  return {
    limit,
    jobSearch: searchInput.value.trim(),
    scheduleId: scheduleSelect.value,
    startDate: startDateInput.value,
    endDate: endDateInput.value,
  }
}

function normalizeDateRange() {
  if (!startDateInput.value || !endDateInput.value) {
    return
  }
  if (startDateInput.value <= endDateInput.value) {
    return
  }

  const swappedStart = endDateInput.value
  endDateInput.value = startDateInput.value
  startDateInput.value = swappedStart
}

function clampLimit(value) {
  const numeric = Number.parseInt(value, 10)
  if (!Number.isFinite(numeric)) {
    return 25
  }
  return Math.min(Math.max(numeric, 1), 200)
}

function buildRequest(filters) {
  if (filters.scheduleId) {
    return {
      path: `/schedules/${encodeURIComponent(filters.scheduleId)}/audits`,
    }
  }

  if (filters.startDate || filters.endDate) {
    const start = filters.startDate ? toStartOfLocalDay(filters.startDate) : new Date(0).toISOString()
    const end = filters.endDate ? toEndOfLocalDay(filters.endDate) : new Date().toISOString()
    return {
      path: `/audits?start=${encodeURIComponent(start)}&end=${encodeURIComponent(end)}`,
    }
  }

  return {
    path: `/audits/recent/${filters.limit}`,
  }
}

function applyClientFilters(audits, filters) {
  const start = filters.startDate ? new Date(toStartOfLocalDay(filters.startDate)).getTime() : null
  const end = filters.endDate ? new Date(toEndOfLocalDay(filters.endDate)).getTime() : null

  return audits
    .filter((audit) => {
      if (filters.scheduleId && audit.scheduleId !== filters.scheduleId) {
        return false
      }

      const startedAt = audit.startTime ? audit.startTime.getTime() : 0
      if (start !== null && startedAt < start) {
        return false
      }
      if (end !== null && startedAt > end) {
        return false
      }
      return true
    })
    .sort((left, right) => right.startTime - left.startTime)
    .slice(0, filters.limit)
}

function filterBySearch(audits, query) {
  if (!query) {
    return audits
  }

  const normalized = query.toLowerCase()
  return audits.filter((audit) => {
    const haystack = [
      audit.displayName,
      audit.scriptPath,
      audit.scheduleId,
    ]
      .join(" ")
      .toLowerCase()

    return haystack.includes(normalized)
  })
}

function normalizeSchedule(schedule) {
  const id = String(schedule.id)
  const scriptPath = schedule.script_path || ""
  const displayName = basename(scriptPath) || `schedule-${id}`

  return {
    id,
    scriptPath,
    displayName,
    label: `${displayName} (#${id})`,
  }
}

function normalizeAudit(audit) {
  const scheduleId = String(audit.schedule_id || "")
  const scriptPath = audit.script_path || scheduleScriptPath(scheduleId)
  const displayName = basename(scriptPath) || scheduleDisplayName(scheduleId)

  return {
    id: String(audit.id || ""),
    scheduleId,
    scriptPath,
    displayName,
    status: audit.status || "unknown",
    output: audit.output || "",
    error: audit.error || "",
    startTime: audit.start_time ? new Date(audit.start_time) : new Date(0),
    endTime: audit.end_time ? new Date(audit.end_time) : new Date(0),
  }
}

function scheduleDisplayName(scheduleId) {
  const schedule = state.schedules.get(scheduleId)
  return schedule ? schedule.displayName : `schedule-${scheduleId || "unknown"}`
}

function scheduleScriptPath(scheduleId) {
  const schedule = state.schedules.get(scheduleId)
  return schedule ? schedule.scriptPath : ""
}

function scheduleLabelById(scheduleId) {
  const schedule = state.schedules.get(scheduleId)
  if (schedule) {
    return schedule.label
  }
  return `schedule #${scheduleId}`
}

function normalizeStatus(status) {
  if (status === "success" || status === "failure") {
    return status
  }
  return "unknown"
}

async function fetchJSON(path, signal) {
  const response = await fetch(path, {
    headers: {
      Accept: "application/json",
    },
    signal,
  })

  if (!response.ok) {
    const message = (await response.text()) || `Request failed with status ${response.status}`
    throw new Error(message)
  }

  return response.json()
}

function setStatus(message, tone) {
  statusNode.textContent = message
  if (tone) {
    statusNode.dataset.tone = tone
    return
  }
  delete statusNode.dataset.tone
}

function buildStatusMessage(filters, count) {
  const target = filters.scheduleId
    ? ` for ${scheduleLabelById(filters.scheduleId)}`
    : ""
  return `Loaded ${count} audit event${count === 1 ? "" : "s"}${target}.`
}

function formatDateRange(startDate, endDate) {
  if (!startDate && !endDate) {
    return ""
  }
  if (startDate && endDate) {
    return ` from ${startDate} to ${endDate}`
  }
  if (startDate) {
    return ` since ${startDate}`
  }
  return ` until ${endDate}`
}

function formatTimestamp(value) {
  if (!(value instanceof Date) || Number.isNaN(value.getTime()) || value.getTime() === 0) {
    return "Unknown"
  }
  return timestampFormat.format(value)
}

function formatRelativeDate(value) {
  if (!(value instanceof Date) || Number.isNaN(value.getTime()) || value.getTime() === 0) {
    return "-"
  }
  return value.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  })
}

function formatDuration(start, end) {
  if (!(start instanceof Date) || !(end instanceof Date)) {
    return "Unknown"
  }

  const ms = end.getTime() - start.getTime()
  if (!Number.isFinite(ms) || ms <= 0) {
    return "<1s"
  }
  if (ms < 1000) {
    return "<1s"
  }
  if (ms < 60000) {
    return `${(ms / 1000).toFixed(ms < 10000 ? 1 : 0)}s`
  }

  const minutes = Math.floor(ms / 60000)
  const seconds = Math.round((ms % 60000) / 1000)
  if (seconds === 0) {
    return `${minutes}m`
  }
  return `${minutes}m ${seconds}s`
}

function truncate(value, maxLength) {
  if (value.length <= maxLength) {
    return value
  }
  return `${value.slice(0, maxLength - 1)}...`
}

function basename(path) {
  if (!path) {
    return ""
  }
  const parts = path.split("/")
  return parts[parts.length - 1] || path
}

function toStartOfLocalDay(dateString) {
  const [year, month, day] = dateString.split("-").map(Number)
  return new Date(year, month - 1, day, 0, 0, 0, 0).toISOString()
}

function toEndOfLocalDay(dateString) {
  const [year, month, day] = dateString.split("-").map(Number)
  return new Date(year, month - 1, day, 23, 59, 59, 999).toISOString()
}
