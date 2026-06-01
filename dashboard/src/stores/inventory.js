import { defineStore } from 'pinia'
import axios from 'axios'

const API = 'http://localhost:8080'

export const useInventoryStore = defineStore('inventory', {
  state: () => ({
    products: [],
    loading: false,
    error: null,
    successMessage: null,
  }),

  getters: {
    // Products that are at or below their low stock threshold
    lowStockProducts: (state) =>
      state.products.filter(p => p.stock <= p.low_stock_threshold),

    // Count products by category number
    totalByCategory: (state) => {
      const counts = { Fiber: 0, LAN: 0, Routers: 0, Switches: 0, Connectors: 0 }
      state.products.forEach(p => {
        const name = categoryName(p.category)
        if (counts[name] !== undefined) counts[name]++
      })
      return counts
    },

    // Total value of all stock
    totalStockValue: (state) =>
      state.products.reduce((sum, p) => sum + p.price * p.stock, 0),
  },

  actions: {
    // Fetch all products from Go backend
    async fetchProducts() {
      this.loading = true
      this.error = null
      try {
        const res = await axios.get(`${API}/products`)
        this.products = res.data || []
      } catch (e) {
        this.error = 'Cannot reach ATX inventory server. Make sure it is running on port 8080.'
      } finally {
        this.loading = false
      }
    },

    // Add a new product
    async addProduct(product) {
      this.error = null
      try {
        await axios.post(`${API}/products`, product)
        this.successMessage = `${product.name} added successfully`
        await this.fetchProducts()
        setTimeout(() => this.successMessage = null, 3000)
      } catch (e) {
        this.error = 'Failed to add product.'
      }
    },

    // Update stock for a product
    async updateStock(id, quantity, reason, note) {
      this.error = null
      try {
        await axios.post(`${API}/stock`, { id, quantity, reason, note })
        this.successMessage = 'Stock updated successfully'
        await this.fetchProducts()
        setTimeout(() => this.successMessage = null, 3000)
      } catch (e) {
        this.error = 'Failed to update stock.'
      }
    },

    // Delete a product
    async deleteProduct(id) {
      this.error = null
      try {
        await axios.delete(`${API}/products/${id}`)
        this.successMessage = 'Product deleted successfully'
        await this.fetchProducts()
        setTimeout(() => this.successMessage = null, 3000)
      } catch (e) {
        this.error = 'Failed to delete product.'
      }
    },

    // Fetch stock history for one product
    async getHistory(id) {
      try {
        const res = await axios.get(`${API}/history/${id}`)
        return res.data || []
      } catch (e) {
        return []
      }
    },
  },
})

// Helper used inside the store
export function categoryName(c) {
  switch (c) {
    case 1: return 'Fiber'
    case 2: return 'LAN'
    case 3: return 'Routers'
    case 4: return 'Switches'
    case 5: return 'Connectors'
    default: return 'Unknown'
  }
}

export function unitName(u) {
  switch (u) {
    case 1: return 'metres'
    case 2: return 'pieces'
    case 3: return 'boxes'
    case 4: return 'rolls'
    default: return 'units'
  }
}

export function categoryNumber(name) {
  switch (name) {
    case 'Fiber': return 1
    case 'LAN': return 2
    case 'Routers': return 3
    case 'Switches': return 4
    case 'Connectors': return 5
    default: return 0
  }
}