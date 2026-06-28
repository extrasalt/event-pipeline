import { useState, useEffect, useCallback } from "react";
import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter, CardHeader } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useCartStore } from "@/stores/cartStore";
import { StarRating } from "@/lib/products";
import { ShoppingCart, Loader2, AlertCircle, RefreshCw } from "lucide-react";

export default function Products() {
  const [products, setProducts] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);
  const addItem = useCartStore((s) => s.addItem);
  const [addingId, setAddingId] = useState(null);

  const load = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/products");
      if (!res.ok) throw new Error("Failed to load products");
      const data = await res.json();
      setProducts(data);
    } catch {
      setError("Failed to load products. Please try again.");
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleAdd = (product) => {
    setAddingId(product.id);
    setTimeout(() => {
      addItem(product);
      setAddingId(null);
    }, 300);
  };

  if (isLoading) {
    return (
      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        <h1 className="mb-8 text-3xl font-bold">Products</h1>
        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <Card key={i} className="overflow-hidden">
              <div className="aspect-square animate-pulse bg-muted" />
              <CardHeader className="p-4 pb-0">
                <div className="h-4 w-3/4 animate-pulse rounded bg-muted" />
                <div className="mt-2 h-3 w-1/4 animate-pulse rounded bg-muted" />
              </CardHeader>
              <CardContent className="p-4 pt-2">
                <div className="h-5 w-1/3 animate-pulse rounded bg-muted" />
              </CardContent>
              <CardFooter className="p-4 pt-0">
                <div className="h-9 w-full animate-pulse rounded bg-muted" />
              </CardFooter>
            </Card>
          ))}
        </div>
      </main>
    );
  }

  if (error) {
    return (
      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        <div className="flex flex-col items-center justify-center gap-4 py-20">
          <AlertCircle className="h-12 w-12 text-destructive" />
          <p className="text-lg text-muted-foreground">{error}</p>
          <Button onClick={load}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Try again
          </Button>
        </div>
      </main>
    );
  }

  if (products.length === 0) {
    return (
      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        <h1 className="mb-8 text-3xl font-bold">Products</h1>
        <div className="flex flex-col items-center justify-center gap-4 py-20">
          <ShoppingCart className="h-12 w-12 text-muted-foreground" />
          <p className="text-lg text-muted-foreground">No products available right now.</p>
          <Button onClick={load}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
        </div>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      <h1 className="mb-8 text-3xl font-bold">Products</h1>
      <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {products.map((product) => (
          <Card key={product.id} className="overflow-hidden">
            <Link to={`/products/${product.id}`} className="block">
              <div className="aspect-square overflow-hidden bg-muted">
                <img
                  src={product.image}
                  alt={product.title}
                  className="h-full w-full object-cover transition-transform hover:scale-105"
                  loading="lazy"
                />
              </div>
              <CardHeader className="p-4 pb-0">
                <h2 className="line-clamp-1 text-base font-semibold hover:text-primary transition-colors">{product.title}</h2>
                <Badge variant="secondary" className="w-fit text-xs">{product.category}</Badge>
              </CardHeader>
              <CardContent className="space-y-1 p-4 pb-2 pt-2">
                <p className="text-xl font-bold">${product.price.toFixed(2)}</p>
                <div className="flex items-center gap-2">
                  <StarRating rate={product.rating.rate} />
                  <span className="text-xs text-muted-foreground">({product.rating.count})</span>
                </div>
              </CardContent>
            </Link>
            <CardFooter className="p-4 pt-0">
              <Button
                className="w-full"
                size="sm"
                onClick={() => handleAdd(product)}
                disabled={addingId === product.id}
                aria-label={`Add ${product.title} to cart`}
              >
                {addingId === product.id ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <ShoppingCart className="mr-2 h-4 w-4" />
                )}
                {addingId === product.id ? "Adding..." : "Add to Cart"}
              </Button>
            </CardFooter>
          </Card>
        ))}
      </div>
    </main>
  );
}
