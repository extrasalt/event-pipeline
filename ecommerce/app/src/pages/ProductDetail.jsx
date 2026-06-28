import { useState, useEffect } from "react";
import { useParams, Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { useCartStore } from "@/stores/cartStore";
import { StarRating } from "@/lib/products";
import { ShoppingCart, ChevronLeft, Loader2, PackageOpen } from "lucide-react";

export default function ProductDetail() {
  const { id } = useParams();
  const addItem = useCartStore((s) => s.addItem);
  const [product, setProduct] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);
  const [quantity, setQuantity] = useState(1);
  const [isAdding, setIsAdding] = useState(false);

  useEffect(() => {
    const load = async () => {
      setIsLoading(true);
      setError(null);
      try {
        const res = await fetch(`/api/products/${id}`);
        if (res.status === 404) {
          setProduct(null);
          return;
        }
        if (!res.ok) throw new Error("Failed to load product");
        setProduct(await res.json());
      } catch {
        setError("Failed to load product. Please try again.");
      } finally {
        setIsLoading(false);
      }
    };
    load();
  }, [id]);

  if (isLoading) {
    return (
      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        <div className="grid gap-10 md:grid-cols-2">
          <div className="aspect-square animate-pulse rounded-xl bg-muted" />
          <div className="space-y-6">
            <div className="h-4 w-20 animate-pulse rounded bg-muted" />
            <div className="h-8 w-3/4 animate-pulse rounded bg-muted" />
            <div className="h-4 w-1/2 animate-pulse rounded bg-muted" />
            <div className="h-10 w-1/4 animate-pulse rounded bg-muted" />
            <div className="h-20 w-full animate-pulse rounded bg-muted" />
          </div>
        </div>
      </main>
    );
  }

  if (error) {
    return (
      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        <div className="flex flex-col items-center justify-center gap-4 py-20">
          <PackageOpen className="h-16 w-16 text-muted-foreground" />
          <h1 className="text-2xl font-bold">Error loading product</h1>
          <p className="text-muted-foreground">{error}</p>
          <Button asChild>
            <Link to="/products">
              <ChevronLeft className="mr-2 h-4 w-4" />
              Back to Products
            </Link>
          </Button>
        </div>
      </main>
    );
  }

  if (!product) {
    return (
      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        <div className="flex flex-col items-center justify-center gap-4 py-20">
          <PackageOpen className="h-16 w-16 text-muted-foreground" />
          <h1 className="text-2xl font-bold">Product not found</h1>
          <p className="text-muted-foreground">The product you&apos;re looking for doesn&apos;t exist.</p>
          <Button asChild>
            <Link to="/products">
              <ChevronLeft className="mr-2 h-4 w-4" />
              Back to Products
            </Link>
          </Button>
        </div>
      </main>
    );
  }

  const handleAdd = () => {
    setIsAdding(true);
    setTimeout(() => {
      addItem(product, quantity);
      setIsAdding(false);
    }, 300);
  };

  return (
    <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      <Button variant="ghost" size="sm" asChild className="mb-6 -ml-3">
        <Link to="/products">
          <ChevronLeft className="mr-1 h-4 w-4" />
          Back to Products
        </Link>
      </Button>

      <div className="grid gap-10 md:grid-cols-2">
        <div className="overflow-hidden rounded-xl bg-muted">
          <img
            src={product.image}
            alt={product.title}
            className="h-full w-full object-cover"
          />
        </div>

        <div className="flex flex-col justify-start gap-6">
          <div className="space-y-3">
            <Badge variant="secondary" className="w-fit text-xs">{product.category}</Badge>
            <h1 className="text-3xl font-bold leading-tight">{product.title}</h1>
            <div className="flex items-center gap-3">
              <StarRating rate={product.rating.rate} />
              <span className="text-sm text-muted-foreground">
                {product.rating.rate} ({product.rating.count} reviews)
              </span>
            </div>
            <p className="text-4xl font-bold text-primary">${product.price.toFixed(2)}</p>
          </div>

          <Separator />

          <p className="text-base leading-relaxed text-muted-foreground">
            {product.description}
          </p>

          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="icon"
                className="h-10 w-10 rounded-full"
                onClick={() => setQuantity(Math.max(1, quantity - 1))}
                aria-label="Decrease quantity"
              >
                −
              </Button>
              <span className="flex h-10 w-12 items-center justify-center text-base font-semibold" aria-live="polite">
                {quantity}
              </span>
              <Button
                variant="outline"
                size="icon"
                className="h-10 w-10 rounded-full"
                onClick={() => setQuantity(quantity + 1)}
                aria-label="Increase quantity"
              >
                +
              </Button>
            </div>
          </div>

          <Button size="lg" className="w-full md:w-auto" onClick={handleAdd} disabled={isAdding}>
            {isAdding ? (
              <Loader2 className="mr-2 h-5 w-5 animate-spin" />
            ) : (
              <ShoppingCart className="mr-2 h-5 w-5" />
            )}
            {isAdding ? "Adding..." : `Add to Cart — $${(product.price * quantity).toFixed(2)}`}
          </Button>
        </div>
      </div>
    </main>
  );
}
