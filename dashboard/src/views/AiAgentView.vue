<template>
  <div class="ai-page">
    <div class="ai-card">
      <div class="ai-header">
        <div>
          <h2>🤖 ATX AI Inventory Agent</h2>
          <p class="ai-sub">Ask anything about your inventory in plain English</p>
        </div>
        <div class="ai-status" :class="connected ? 'connected' : 'disconnected'">
          {{ connected ? '● Connected' : '○ Server offline' }}
        </div>
      </div>

      <!-- Suggested questions -->
      <div class="suggestions">
        <span class="sug-label">Try asking:</span>
        <button v-for="q in suggestions" :key="q" class="sug-btn" @click="ask(q)">{{ q }}</button>
      </div>

      <!-- Chat messages -->
      <div class="chat-window" ref="chatWindow">
        <div v-if="messages.length === 0" class="chat-empty">
          No messages yet. Ask a question below.
        </div>
        <div v-for="(m, i) in messages" :key="i" class="message" :class="m.role">
          <div class="message-bubble">
            <div class="message-role">{{ m.role === 'user' ? 'You' : 'ATX AI' }}</div>
            <div class="message-text" v-html="formatMessage(m.text)"></div>
            <div v-if="m.tool" class="tool-call">🔧 Called tool: {{ m.tool }}</div>
          </div>
        </div>
        <div v-if="thinking" class="message ai">
          <div class="message-bubble">
            <div class="message-role">ATX AI</div>
            <div class="thinking">Thinking<span class="dots">...</span></div>
          </div>
        </div>
      </div>

      <!-- Input -->
      <div class="chat-input">
        <input
          v-model="question"
          @keydown.enter="sendQuestion"
          placeholder="e.g. Which products are running low on stock?"
          :disabled="thinking"
        />
        <button class="btn-send" @click="sendQuestion" :disabled="thinking || !question.trim()">
          {{ thinking ? '...' : 'Ask' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick, onMounted } from 'vue'
import axios from 'axios'

const question = ref('')
const messages = ref([])
const thinking = ref(false)
const connected = ref(false)
const chatWindow = ref(null)

const suggestions = [
  'Which products are running low?',
  'Give me an inventory summary',
  'What is the price of ATX-006?',
  'How many fiber products do we have?',
]

onMounted(async () => {
  try {
    await axios.get('http://localhost:8080/products')
    connected.value = true
  } catch {
    connected.value = false
  }
})

function formatMessage(text) {
  return text
    .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
    .replace(/\n/g, '<br>')
    .replace(/•/g, '&bull;')
}

async function scrollToBottom() {
  await nextTick()
  if (chatWindow.value) {
    chatWindow.value.scrollTop = chatWindow.value.scrollHeight
  }
}

async function ask(q) {
  question.value = q
  await sendQuestion()
}

async function sendQuestion() {
  const q = question.value.trim()
  if (!q || thinking.value) return

  messages.value.push({ role: 'user', text: q })
  question.value = ''
  thinking.value = true
  await scrollToBottom()

  try {
    const res = await axios.post('http://localhost:8080/ai', { question: q })
    messages.value.push({
      role: 'ai',
      text: res.data.answer,
      tool: res.data.tool_used,
    })
  } catch (e) {
    messages.value.push({
      role: 'ai',
      text: 'Error reaching the AI agent. Make sure the server is running and your API key is set.',
    })
  }

  thinking.value = false
  await scrollToBottom()
}
</script>

<style scoped>
.ai-page { max-width: 860px; margin: 0 auto; }

.ai-card {
  background: white;
  border-radius: 16px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.08);
  overflow: hidden;
}

.ai-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 1.5rem 1.5rem 1rem;
  border-bottom: 1px solid #f0f0f0;
}
.ai-header h2 { font-size: 18px; font-weight: 600; }
.ai-sub { font-size: 13px; color: #888; margin-top: 4px; }

.ai-status {
  font-size: 12px;
  font-weight: 600;
  padding: 4px 10px;
  border-radius: 20px;
}
.connected { background: #f0fff4; color: #276749; }
.disconnected { background: #fff5f5; color: #e53e3e; }

.suggestions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  padding: 0.75rem 1.5rem;
  background: #fafbff;
  border-bottom: 1px solid #f0f0f0;
}
.sug-label { font-size: 12px; color: #888; font-weight: 500; white-space: nowrap; }
.sug-btn {
  font-size: 12px;
  padding: 4px 12px;
  border-radius: 20px;
  border: 1.5px solid #e2e8f0;
  background: white;
  color: #555;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}
.sug-btn:hover { border-color: #0070f3; color: #0070f3; background: #f0f7ff; }

.chat-window {
  height: 460px;
  overflow-y: auto;
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.chat-empty {
  text-align: center;
  color: #aaa;
  font-size: 14px;
  margin: auto;
}

.message { display: flex; }
.message.user { justify-content: flex-end; }
.message.ai { justify-content: flex-start; }

.message-bubble {
  max-width: 75%;
  padding: 12px 16px;
  border-radius: 16px;
}
.message.user .message-bubble {
  background: #0070f3;
  color: white;
  border-bottom-right-radius: 4px;
}
.message.ai .message-bubble {
  background: #f4f6f9;
  color: #1a1a2e;
  border-bottom-left-radius: 4px;
}

.message-role {
  font-size: 11px;
  font-weight: 600;
  margin-bottom: 4px;
  opacity: 0.7;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.message-text { font-size: 14px; line-height: 1.6; }

.tool-call {
  font-size: 11px;
  margin-top: 8px;
  padding: 4px 8px;
  background: rgba(0,0,0,0.06);
  border-radius: 6px;
  color: #555;
}

.thinking { font-size: 14px; color: #888; }
.dots { animation: blink 1.2s infinite; }
@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

.chat-input {
  display: flex;
  gap: 8px;
  padding: 1rem 1.5rem;
  border-top: 1px solid #f0f0f0;
  background: #fafbff;
}
.chat-input input {
  flex: 1;
  padding: 10px 16px;
  border: 1.5px solid #e2e8f0;
  border-radius: 10px;
  font-size: 14px;
  outline: none;
}
.chat-input input:focus { border-color: #0070f3; }
.chat-input input:disabled { background: #f4f4f4; }

.btn-send {
  padding: 10px 20px;
  background: #0070f3;
  color: white;
  border: none;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}
.btn-send:hover:not(:disabled) { background: #0060d3; }
.btn-send:disabled { opacity: 0.5; cursor: not-allowed; }
</style>