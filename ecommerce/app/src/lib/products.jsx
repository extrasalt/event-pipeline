import data from "./products.json";

export const PRODUCTS = data;

export function getProductById(id) {
  return PRODUCTS.find((p) => p.id === id) || null;
}

export function StarRating({ rate }) {
  return (
    <span className="text-yellow-500 text-sm" aria-label={`Rating: ${rate} out of 5`}>
      {"★".repeat(Math.round(rate))}{"☆".repeat(5 - Math.round(rate))}
    </span>
  );
}
