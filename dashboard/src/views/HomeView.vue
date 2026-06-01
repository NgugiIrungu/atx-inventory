<template>
  <div>
    <!-- Stats row -->
    <div class="stats-row">
      <div class="stat-card">
        <div class="stat-val">{{ store.products.length }}</div>
        <div class="stat-label">Total Products</div>
      </div>
      <div class="stat-card">
        <div class="stat-val">${{ store.totalStockValue.toLocaleString('en', {minimumFractionDigits: 2, maximumFractionDigits: 2}) }}</div>
        <div class="stat-label">Total Stock Value</div>
      </div>
      <div class="stat-card" :class="{ danger: store.lowStockProducts.length > 0 }">
        <div class="stat-val">{{ store.lowStockProducts.length }}</div>
        <div class="stat-label">Low Stock Alerts</div>
      </div>
      <div class="stat-card">
        <div class="stat-val">{{ Object.keys(store.totalByCategory).filter(k => store.totalByCategory[k] > 0).length }}</div>
        <div class="stat-label">Categories</div>
      </div>
    </div>

    <!-- Success / Error messages -->
    <div v-if="store.successMessage" class="alert success">✓ {{ store.successMessage }}</div>
    <div v-if="store.error" class="alert error">⚠ {{ store.error }}</div>

    <!-- Add Product Form -->
    <div class="card">
      <div class="card-header" @click="showForm = !showForm">
        <h2>➕ Add New Product</h2>
        <span>{{ showForm ? '▲' : '▼' }}</span>
      </div>
      <div v-if="showForm" class="card-body">
        <div class="form-grid">
          <div class="form-group">
            <label>Product Name</label>
            <input v-model="form.name" placeholder="e.g. Cat7 LAN Cable" />
          </div>
          <div class="form-group">
            <label>Description</label>
            <input v-model="form.description" placeholder="Brief description" />
          </div>
          <div class="form-group">
            <label>Category</label>
            <select v-model="form.category">
              <option value="1">Fiber</option>
              <option value="2">LAN</option>
              <option value="3">Routers</option>
              <option value="4">Switches</option>
              <option value="5">Connectors</option>
            </select>
          </div>
          <div class="form-group">
            <label>Unit</label>
            <select v-model="form.unit">
              <option value="1">Metres</option>
              <option value="2">Pieces</option>
              <option value="3">Boxes</option>
              <option value="4">Rolls</option>
            </select>
          </div>
          <div class="form-group">
            <label>Price ($)</label>
            <input v-model="form.price" type="number" step="0.01" placeholder="0.00" />
          </div>
          <div class="form-group">
            <label>Initial Stock</label>
            <input v-model="form.stock" type="number" placeholder="0" />
          </div>
          <div class="form-group">
            <label>Low Stock Threshold</label>
            <input v-model="form.low_stock_threshold" type="number" placeholder="10" />
          </div>
        </div>
        <button class="btn btn-primary" @click="addProduct">Add Product</button>
      </div>
    </div>

    <!-- Product Table -->
    <div class="card">
      <div class="card-header">
        <h2>📦 Inventory</h2>
        <div style="display:flex; gap:8px; align-items:center">
          <input v-model="search" class="search" placeholder="Search products..." />
          <button class="btn btn-secondary" @click="store.fetchProducts()">↻ Refresh</button>
        </div>
      </div>
      <div class="card-body" style="padding:0">
        <div v-if="store.loading" class="empty">Loading inventory...</div>
        <div v-else-if="filteredProducts.length === 0" class="empty">No products found.</div>
        <table v-else class="table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Name</th>
              <th>Category</th>
              <th>Price</th>
              <th>Stock</th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in filteredProducts" :key="p.id" :class="{ 'low-row': p.stock <= p.low_stock_threshold }">
              <td class="mono">{{ p.id }}</td>
              <td>
                <div class="product-name">{{ p.name }}</div>
                <div class="product-desc">{{ p.description }}</div>
              </td>
              <td><span class="badge" :class="badgeClass(p.category)">{{ categoryName(p.category) }}</span></td>
              <td class="mono">${{ p.price.toFixed(2) }}</td>
              <td class="mono">{{ p.stock.toLocaleString() }} {{ unitName(p.unit) }}</td>
              <td>
                <span class="status" :class="p.stock <= p.low_stock_threshold ? 'status-low' : 'status-ok'">
                  {{ p.stock <= p.low_stock_threshold ? '⚠ LOW' : '✓ OK' }}
                </span>
              </td>
              <td>
                <div style="display:flex; gap:6px">
                  <button class="btn btn-sm btn-secondary" @click="openStock(p)">Update Stock</button>
                  <button class="btn btn-sm btn-history" @click="openHistory(p)">History</button>
                  <button class="btn btn-sm btn-danger" @click="deleteProduct(p)">Delete</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Update Stock Modal -->
    <div v-if="stockModal" class="modal-overlay" @click.self="stockModal = false">
      <div class="modal">
        <h3>Update Stock — {{ selectedProduct?.name }}</h3>
        <p class="modal-sub">Current stock: {{ selectedProduct?.stock }} {{ unitName(selectedProduct?.unit) }}</p>
        <div class="form-group">
          <label>Change Amount (+ to add, - to remove)</label>
          <input v-model="stockChange" type="number" placeholder="e.g. 500 or -10" />
        </div>
        <div class="form-group">
          <label>Reason</label>
          <select v-model="stockReason">
            <option value="1">Restock</option>
            <option value="2">Sale</option>
            <option value="3">Damage</option>
            <option value="4">Manual Adjustment</option>
          </select>
        </div>
        <div class="form-group">
          <label>Note</label>
          <input v-model="stockNote" placeholder="e.g. Supplier delivery INV-2024-001" />
        </div>
        <div style="display:flex; gap:8px; margin-top:1rem">
          <button class="btn btn-primary" @click="submitStock">Confirm</button>
          <button class="btn btn-secondary" @click="stockModal = false">Cancel</button>
        </div>
      </div>
    </div>

    <!-- History Modal -->
    <div v-if="historyModal" class="modal-overlay" @click.self="historyModal = false">
      <div class="modal modal-wide">
        <h3>Stock History — {{ selectedProduct?.name }}</h3>
        <div v-if="historyLoading" class="empty">Loading history...</div>
        <div v-else-if="history.length === 0" class="empty">No history recorded yet.</div>
        <table v-else class="table">
          <thead>
            <tr>
              <th>Timestamp</th>
              <th>Change</th>
              <th>Stock After</th>
              <th>Reason</th>
              <th>Note</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(h, i) in history" :key="i">
              <td class="mono">{{ h.timestamp }}</td>
              <td class="mono" :class="h.quantity > 0 ? 'positive' : 'negative'">
                {{ h.quantity > 0 ? '+' : '' }}{{ h.quantity }}
              </td>
              <td class="mono">{{ h.stock_after }}</td>
              <td>{{ reasonName(h.reason) }}</td>
              <td>{{ h.note }}</td>
            </tr>
          </tbody>
        </table>
        <button class="btn btn-secondary" style="margin-top:1rem" @click="historyModal = false">Close</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useInventoryStore, categoryName, unitName } from '../stores/inventory'

const store = useInventoryStore()
const search = ref('')
const showForm = ref(false)

// Add product form
const form = ref({
  name: '', description: '', category: '1',
  unit: '1', price: '', stock: '', low_stock_threshold: '10'
})

// Stock update modal
const stockModal = ref(false)
const selectedProduct = ref(null)
const stockChange = ref('')
const stockReason = ref('1')
const stockNote = ref('')

// History modal
const historyModal = ref(false)
const history = ref([])
const historyLoading = ref(false)

const filteredProducts = computed(() =>
  store.products.filter(p =>
    p.name.toLowerCase().includes(search.value.toLowerCase()) ||
    p.id.toLowerCase().includes(search.value.toLowerCase())
  )
)

function badgeClass(c) {
  return { 1: 'badge-fiber', 2: 'badge-lan', 3: 'badge-router', 4: 'badge-switch', 5: 'badge-connector' }[c] || ''
}

function reasonName(r) {
  return { 1: 'Restock', 2: 'Sale', 3: 'Damage', 4: 'Adjustment' }[r] || 'Unknown'
}

async function addProduct() {
  if (!form.value.name || !form.value.price || !form.value.stock) {
    alert('Please fill in name, price, and stock.')
    return
  }
  await store.addProduct({
    name: form.value.name,
    description: form.value.description,
    category: parseInt(form.value.category),
    unit: parseInt(form.value.unit),
    price: parseFloat(form.value.price),
    stock: parseInt(form.value.stock),
    low_stock_threshold: parseInt(form.value.low_stock_threshold),
  })
  form.value = { name: '', description: '', category: '1', unit: '1', price: '', stock: '', low_stock_threshold: '10' }
  showForm.value = false
}

function openStock(p) {
  selectedProduct.value = p
  stockChange.value = ''
  stockNote.value = ''
  stockReason.value = '1'
  stockModal.value = true
}

async function submitStock() {
  if (!stockChange.value) { alert('Enter a quantity.'); return }
  await store.updateStock(selectedProduct.value.id, parseInt(stockChange.value), parseInt(stockReason.value), stockNote.value)
  stockModal.value = false
}

async function openHistory(p) {
  selectedProduct.value = p
  historyModal.value = true
  historyLoading.value = true
  history.value = await store.getHistory(p.id)
  historyLoading.value = false
}

async function deleteProduct(p) {
  if (confirm(`Delete ${p.name}? This cannot be undone.`)) {
    await store.deleteProduct(p.id)
  }
}

onMounted(() => store.fetchProducts())
</script>

<style scoped>
.stats-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 1rem; margin-bottom: 1.5rem; }
.stat-card { background: white; border-radius: 12px; padding: 1.25rem 1.5rem; box-shadow: 0 1px 4px rgba(0,0,0,0.08); }
.stat-card.danger { border-left: 4px solid #e53e3e; }
.stat-val { font-size: 28px; font-weight: 700; color: #1a1a2e; }
.stat-label { font-size: 13px; color: #666; margin-top: 4px; }

.alert { padding: 12px 16px; border-radius: 8px; margin-bottom: 1rem; font-size: 14px; font-weight: 500; }
.alert.success { background: #f0fff4; color: #276749; border: 1px solid #9ae6b4; }
.alert.error { background: #fff5f5; color: #c53030; border: 1px solid #fed7d7; }

.card { background: white; border-radius: 12px; box-shadow: 0 1px 4px rgba(0,0,0,0.08); margin-bottom: 1.5rem; overflow: hidden; }
.card-header { display: flex; align-items: center; justify-content: space-between; padding: 1rem 1.5rem; border-bottom: 1px solid #f0f0f0; cursor: pointer; }
.card-header h2 { font-size: 16px; font-weight: 600; }
.card-body { padding: 1.5rem; }

.form-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; margin-bottom: 1rem; }
.form-group { display: flex; flex-direction: column; gap: 4px; }
.form-group label { font-size: 12px; font-weight: 600; color: #555; text-transform: uppercase; letter-spacing: 0.04em; }
.form-group input, .form-group select { padding: 8px 12px; border: 1.5px solid #e2e8f0; border-radius: 8px; font-size: 14px; outline: none; transition: border 0.2s; }
.form-group input:focus, .form-group select:focus { border-color: #0070f3; }

.search { padding: 7px 12px; border: 1.5px solid #e2e8f0; border-radius: 8px; font-size: 14px; outline: none; width: 220px; }
.search:focus { border-color: #0070f3; }

.table { width: 100%; border-collapse: collapse; font-size: 14px; }
.table th { background: #f8fafc; padding: 10px 16px; text-align: left; font-size: 12px; font-weight: 600; color: #666; text-transform: uppercase; letter-spacing: 0.04em; border-bottom: 1px solid #f0f0f0; }
.table td { padding: 12px 16px; border-bottom: 1px solid #f8fafc; }
.table tr:last-child td { border-bottom: none; }
.table tr:hover td { background: #fafbff; }
.low-row td { background: #fff8f8 !important; }

.product-name { font-weight: 500; }
.product-desc { font-size: 12px; color: #888; margin-top: 2px; }
.mono { font-family: monospace; }
.positive { color: #276749; font-weight: 600; }
.negative { color: #c53030; font-weight: 600; }

.badge { font-size: 11px; font-weight: 600; padding: 3px 8px; border-radius: 20px; }
.badge-fiber { background: #e9d8fd; color: #553c9a; }
.badge-lan { background: #bee3f8; color: #2c5282; }
.badge-router { background: #fefcbf; color: #744210; }
.badge-switch { background: #c6f6d5; color: #276749; }
.badge-connector { background: #fed7d7; color: #9b2c2c; }

.status { font-size: 12px; font-weight: 600; padding: 3px 8px; border-radius: 20px; }
.status-ok { background: #f0fff4; color: #276749; }
.status-low { background: #fff5f5; color: #e53e3e; }

.empty { padding: 3rem; text-align: center; color: #888; font-size: 14px; }

.btn { padding: 8px 16px; border-radius: 8px; border: none; font-size: 14px; font-weight: 500; cursor: pointer; transition: all 0.2s; }
.btn-primary { background: #0070f3; color: white; }
.btn-primary:hover { background: #0060d3; }
.btn-secondary { background: #f0f0f0; color: #333; }
.btn-secondary:hover { background: #e0e0e0; }
.btn-danger { background: #fff5f5; color: #e53e3e; }
.btn-danger:hover { background: #fed7d7; }
.btn-history { background: #ebf8ff; color: #2b6cb0; }
.btn-history:hover { background: #bee3f8; }
.btn-sm { padding: 5px 10px; font-size: 12px; }

.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: white; border-radius: 16px; padding: 2rem; width: 480px; max-height: 90vh; overflow-y: auto; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
.modal-wide { width: 780px; }
.modal h3 { font-size: 18px; font-weight: 600; margin-bottom: 4px; }
.modal-sub { font-size: 13px; color: #666; margin-bottom: 1.5rem; }
</style>