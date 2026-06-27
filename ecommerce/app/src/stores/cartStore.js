import { create } from "zustand";

const loadCart = () => {
  try {
    const c = localStorage.getItem("cart");
    return c ? JSON.parse(c) : [];
  } catch {
    return [];
  }
};

const persist = (items) => {
  localStorage.setItem("cart", JSON.stringify(items));
};

export const useCartStore = create((set, get) => ({
  items: loadCart(),

  addItem: (product, quantity = 1) => {
    const items = [...get().items];
    const idx = items.findIndex((i) => i.product.id === product.id);
    if (idx >= 0) {
      items[idx] = { ...items[idx], quantity: items[idx].quantity + quantity };
    } else {
      items.push({ product, quantity });
    }
    persist(items);
    set({ items });
  },

  removeItem: (productId) => {
    const items = get().items.filter((i) => i.product.id !== productId);
    persist(items);
    set({ items });
  },

  updateQuantity: (productId, quantity) => {
    if (quantity <= 0) {
      return get().removeItem(productId);
    }
    const items = get().items.map((i) =>
      i.product.id === productId ? { ...i, quantity } : i
    );
    persist(items);
    set({ items });
  },

  clearCart: () => {
    persist([]);
    set({ items: [] });
  },

  get totalItems() {
    return get().items.reduce((sum, i) => sum + i.quantity, 0);
  },

  get subtotal() {
    return get().items.reduce((sum, i) => sum + i.product.price * i.quantity, 0);
  },
}));
