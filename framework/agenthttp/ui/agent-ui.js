(() => {
  "use strict";

  const $ = (selector) => document.querySelector(selector);
  const state = { sessionId: "", running: false, abortController: null, toastTimer: null };
  const els = {
    transcript: $("#transcript"), emptyState: $("#emptyState"), messageForm: $("#messageForm"),
    messageInput: $("#messageInput"), sendButton: $("#sendButton"), stopButton: $("#stopButton"),
    composerHint: $("#composerHint"), newSessionButton: $("#newSessionButton"), sessionId: $("#sessionId"),
    copySessionButton: $("#copySessionButton"), configForm: $("#configForm"), baseUrl: $("#baseUrl"),
    model: $("#model"), apiKey: $("#apiKey"), saveConfigButton: $("#saveConfigButton"),
    clearTokenButton: $("#clearTokenButton"), toggleSecretButton: $("#toggleSecretButton"),
    statusDot: $("#statusDot"), statusText: $("#statusText"), tokenBadge: $("#tokenBadge"),
    modelPill: $("#modelPill"), toolList: $("#toolList"), toolCount: $("#toolCount"), toast: $("#toast"),
    sidebar: $("#sidebar"), sidebarToggle: $("#sidebarToggle"), sidebarBackdrop: $("#sidebarBackdrop")
  };

  function toast(message, isError = false) {
    clearTimeout(state.toastTimer);
    els.toast.textContent = message;
    els.toast.className = `toast visible${isError ? " error" : ""}`;
    state.toastTimer = setTimeout(() => { els.toast.className = "toast"; }, 2800);
  }

  async function api(path, options = {}) {
    const response = await fetch(path, options);
    const contentType = response.headers.get("content-type") || "";
    const data = contentType.includes("application/json") ? await response.json() : null;
    if (!response.ok) throw new Error(data?.error || `${response.status} ${response.statusText}`);
    return data;
  }

  function updateConnection(ok, text) {
    els.statusDot.className = `status-dot ${ok ? "online" : "error"}`;
    els.statusText.textContent = text;
  }

  function applyConfigView(config) {
    els.baseUrl.value = config.base_url || "";
    els.model.value = config.model || "";
    els.modelPill.textContent = config.model || "未选择模型";
    els.tokenBadge.textContent = config.api_key_configured ? "Token 已就绪" : "未配置 Token";
    els.tokenBadge.className = `secure-badge${config.api_key_configured ? " ready" : ""}`;
    els.apiKey.value = "";
  }

  async function loadConfig() {
    try {
      const config = await api("/config", { cache: "no-store" });
      applyConfigView(config);
      updateConnection(true, "Agent 服务已连接");
    } catch (error) {
      updateConnection(false, "Agent 服务不可用");
      toast(error.message, true);
    }
  }

  async function loadTools() {
    try {
      const data = await api("/tools", { cache: "no-store" });
      const tools = data.tools || [];
      els.toolCount.textContent = String(tools.length);
      els.toolList.replaceChildren(...tools.map((tool) => {
        const node = document.createElement("span");
        node.className = "tool-chip";
        node.textContent = tool.name;
        node.title = tool.description || tool.name;
        return node;
      }));
      if (!tools.length) els.toolList.innerHTML = '<span class="muted">当前没有注册工具</span>';
    } catch (error) {
      els.toolList.innerHTML = '<span class="muted">工具读取失败</span>';
    }
  }

  function setSession(id) {
    state.sessionId = id;
    els.sessionId.textContent = id || "尚未创建";
    els.sessionId.title = id || "";
    if (id) sessionStorage.setItem("hade.agent.session", id);
    else sessionStorage.removeItem("hade.agent.session");
  }

  function resetTranscript() {
    els.transcript.replaceChildren(els.emptyState);
    els.emptyState.hidden = false;
  }

  async function createSession(showNotice = true, reset = true, allowWhileRunning = false) {
    if (state.running && !allowWhileRunning) return false;
    try {
      els.newSessionButton.disabled = true;
      const data = await api("/sessions", { method: "POST" });
      setSession(data.id);
      if (reset) resetTranscript();
      if (showNotice) toast("新会话已创建");
      return true;
    } catch (error) {
      toast(`创建会话失败：${error.message}`, true);
      return false;
    } finally {
      els.newSessionButton.disabled = false;
    }
  }

  function messageNode(role, content) {
    const turn = document.createElement("article");
    turn.className = "turn";
    const label = document.createElement("div");
    label.className = "turn-label";
    label.textContent = role === "user" ? "You" : role === "tool" ? "Tool result" : "Hade Agent";
    const message = document.createElement("div");
    message.className = role === "user" ? "user-message" : "agent-card";
    if (role === "user") message.textContent = content;
    else {
      const answer = document.createElement("div");
      answer.className = "agent-answer";
      answer.textContent = content;
      message.appendChild(answer);
    }
    turn.append(label, message);
    return turn;
  }

  function addUserMessage(content) {
    els.emptyState.hidden = true;
    const node = messageNode("user", content);
    els.transcript.appendChild(node);
    return node;
  }

  function addAgentRun() {
    const turn = messageNode("assistant", "Agent 正在处理");
    const card = turn.querySelector(".agent-card");
    const answer = turn.querySelector(".agent-answer");
    answer.classList.add("pending", "typing");
    const log = document.createElement("div");
    log.className = "event-log";
    card.appendChild(log);
    els.transcript.appendChild(turn);
    return { turn, answer, log, finalReceived: false };
  }

  function addEvent(run, type, label, content) {
    const row = document.createElement(type === "thought" ? "details" : "div");
    row.className = `event-row ${type}`;
    const kind = document.createElement("div");
    kind.className = "event-kind";
    kind.textContent = label;
    const value = document.createElement("div");
    value.className = "event-content";
    value.textContent = content;
    if (type === "thought") {
      const summary = document.createElement("summary");
      summary.className = "event-summary";
      const hint = document.createElement("span");
      hint.className = "event-toggle-hint";
      hint.textContent = "点击展开";
      summary.append(kind, hint);
      row.append(summary, value);
    } else {
      row.append(kind, value);
    }
    run.log.appendChild(row);
  }

  function stringify(value) {
    if (typeof value === "string") return value;
    try { return JSON.stringify(value, null, 2); } catch (_) { return String(value); }
  }

  function handleEvent(run, event, data) {
    if (event === "thought") {
      if (data.content) addEvent(run, "thought", "Thought", data.content);
    } else if (event === "action") {
      addEvent(run, "action", `Action · ${data.name || "tool"}`, stringify(data.arguments || {}));
    } else if (event === "observation") {
      addEvent(run, "observation", `Result · ${data.name || "tool"}`, stringify(data.content || ""));
    } else if (event === "final") {
      run.finalReceived = true;
      run.answer.textContent = data.content || "（模型返回了空内容）";
      run.answer.classList.remove("pending", "typing");
    } else if (event === "error") {
      addEvent(run, "error", "Error", data.message || data.code || "unknown error");
      if (!run.finalReceived) {
        run.answer.textContent = `执行失败：${data.message || data.code || "unknown error"}`;
        run.answer.classList.add("error-text");
        run.answer.classList.remove("pending", "typing");
      }
    }
  }

  async function consumeSSE(response, run) {
    if (!response.body) throw new Error("浏览器无法读取事件流");
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    while (true) {
      const { value, done } = await reader.read();
      buffer += decoder.decode(value || new Uint8Array(), { stream: !done }).replace(/\r\n/g, "\n");
      let boundary;
      while ((boundary = buffer.indexOf("\n\n")) >= 0) {
        const block = buffer.slice(0, boundary);
        buffer = buffer.slice(boundary + 2);
        let event = "message";
        const dataLines = [];
        for (const line of block.split("\n")) {
          if (line.startsWith("event:")) event = line.slice(6).trim();
          if (line.startsWith("data:")) dataLines.push(line.slice(5).trimStart());
        }
        if (dataLines.length) {
          let data = {};
          try { data = JSON.parse(dataLines.join("\n")); } catch (_) { data = { content: dataLines.join("\n") }; }
          handleEvent(run, event, data);
          scrollToBottom();
        }
      }
      if (done) break;
    }
  }

  function setRunning(running) {
    state.running = running;
    els.sendButton.disabled = running;
    els.newSessionButton.disabled = running;
    els.stopButton.hidden = !running;
    els.composerHint.textContent = running ? "Agent 正在执行…" : "SSE 实时事件流";
  }

  function scrollToBottom() {
    window.requestAnimationFrame(() => window.scrollTo({ top: document.body.scrollHeight, behavior: "smooth" }));
  }

  function postMessage(message, signal) {
    return fetch(`/sessions/${encodeURIComponent(state.sessionId)}/messages`, {
      method: "POST",
      headers: { "Content-Type": "application/json", "Accept": "text/event-stream" },
      body: JSON.stringify({ message }),
      signal
    });
  }

  async function postMessageWithRecovery(message, signal) {
    let response;
    try {
      response = await postMessage(message, signal);
    } catch (error) {
      if (error.name === "AbortError") throw error;
      // 开发时 Agent 经常会重启。短暂断连后创建一个新内存 Session，再重试一次。
      await new Promise((resolve) => setTimeout(resolve, 350));
      setSession("");
      if (!await createSession(false, false, true)) throw error;
      toast("Agent 已重新连接，已创建新会话");
      return postMessage(message, signal);
    }

    if (response.status !== 404) return response;
    const data = await response.clone().json().catch(() => ({}));
    if (data.error !== "session_not_found") return response;

    setSession("");
    if (!await createSession(false, false, true)) return response;
    toast("旧 Session 已失效，已自动创建新会话");
    return postMessage(message, signal);
  }

  async function sendMessage(message) {
    if (state.running || !message.trim()) return;
    if (!state.sessionId) await createSession(false);
    if (!state.sessionId) return;

    addUserMessage(message.trim());
    const run = addAgentRun();
    els.messageInput.value = "";
    resizeInput();
    setRunning(true);
    scrollToBottom();
    state.abortController = new AbortController();

    try {
      const response = await postMessageWithRecovery(message.trim(), state.abortController.signal);
      if (!response.ok) {
        const data = await response.json().catch(() => ({}));
        throw new Error(data.error || `${response.status} ${response.statusText}`);
      }
      await consumeSSE(response, run);
      if (!run.finalReceived && !run.answer.classList.contains("error-text")) {
        run.answer.textContent = "执行已结束，没有最终回复。";
        run.answer.classList.remove("pending", "typing");
      }
    } catch (error) {
      const aborted = error.name === "AbortError";
      run.answer.textContent = aborted ? "已停止这次执行。" : `请求失败：${error.message}`;
      run.answer.className = `agent-answer${aborted ? "" : " error-text"}`;
      if (!aborted) toast(error.message, true);
    } finally {
      state.abortController = null;
      setRunning(false);
      els.messageInput.focus();
    }
  }

  async function restoreSession() {
    const id = sessionStorage.getItem("hade.agent.session");
    if (!id) return createSession(false);
    try {
      const session = await api(`/sessions/${encodeURIComponent(id)}`, { cache: "no-store" });
      setSession(session.id);
      const messages = session.messages || [];
      if (!messages.length) return;
      els.emptyState.hidden = true;
      for (const message of messages) {
        if (message.role === "user" || (message.role === "assistant" && !message.tool_calls?.length)) {
          els.transcript.appendChild(messageNode(message.role, message.content || ""));
        }
      }
    } catch (_) {
      setSession("");
      await createSession(false);
    }
  }

  function resizeInput() {
    els.messageInput.style.height = "auto";
    els.messageInput.style.height = `${Math.min(els.messageInput.scrollHeight, 180)}px`;
  }

  els.configForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    const payload = { base_url: els.baseUrl.value.trim(), model: els.model.value.trim() };
    if (els.apiKey.value) payload.api_key = els.apiKey.value;
    try {
      els.saveConfigButton.disabled = true;
      els.saveConfigButton.textContent = "正在应用…";
      const config = await api("/config", {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload)
      });
      applyConfigView(config);
      updateConnection(true, "Agent 服务已连接");
      toast("模型配置已应用到当前进程");
    } catch (error) {
      toast(`配置失败：${error.message}`, true);
    } finally {
      els.saveConfigButton.disabled = false;
      els.saveConfigButton.textContent = "应用配置";
    }
  });

  els.clearTokenButton.addEventListener("click", async () => {
    try {
      const config = await api("/config", {
        method: "PUT", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ base_url: els.baseUrl.value.trim(), model: els.model.value.trim(), api_key: "" })
      });
      applyConfigView(config);
      toast("当前进程中的 Token 已清除");
    } catch (error) { toast(error.message, true); }
  });

  els.messageForm.addEventListener("submit", (event) => { event.preventDefault(); sendMessage(els.messageInput.value); });
  els.messageInput.addEventListener("input", resizeInput);
  els.messageInput.addEventListener("keydown", (event) => {
    if (event.key === "Enter" && !event.shiftKey && !event.isComposing) {
      event.preventDefault();
      els.messageForm.requestSubmit();
    }
  });
  els.stopButton.addEventListener("click", () => state.abortController?.abort());
  els.newSessionButton.addEventListener("click", () => createSession(true));
  els.copySessionButton.addEventListener("click", async () => {
    if (!state.sessionId) return;
    await navigator.clipboard.writeText(state.sessionId);
    toast("Session ID 已复制");
  });
  els.toggleSecretButton.addEventListener("click", () => {
    const reveal = els.apiKey.type === "password";
    els.apiKey.type = reveal ? "text" : "password";
    els.toggleSecretButton.textContent = reveal ? "隐藏" : "显示";
  });
  document.querySelectorAll("[data-base]").forEach((button) => button.addEventListener("click", () => {
    els.baseUrl.value = button.dataset.base;
    els.model.value = button.dataset.model;
  }));
  document.querySelectorAll("[data-prompt]").forEach((button) => button.addEventListener("click", () => sendMessage(button.dataset.prompt)));

  function toggleSidebar(open) {
    els.sidebar.classList.toggle("open", open);
    els.sidebarBackdrop.classList.toggle("open", open);
  }
  els.sidebarToggle.addEventListener("click", () => toggleSidebar(true));
  els.sidebarBackdrop.addEventListener("click", () => toggleSidebar(false));

  Promise.all([loadConfig(), loadTools(), restoreSession()]);
})();
