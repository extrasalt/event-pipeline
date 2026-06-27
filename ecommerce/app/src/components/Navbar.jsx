import { Link, useNavigate } from "react-router-dom";
import { ShoppingCart, User, LogOut, Menu, X } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useAuthStore } from "@/stores/authStore";
import { useCartStore } from "@/stores/cartStore";

export default function Navbar() {
  const { user, logout } = useAuthStore();
  const totalItems = useCartStore((s) => s.items.reduce((sum, i) => sum + i.quantity, 0));
  const navigate = useNavigate();
  const [menuOpen, setMenuOpen] = useState(false);

  const handleLogout = async () => {
    await logout();
    navigate("/login");
  };

  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
        <Link to="/products" className="text-xl font-bold tracking-tight">
          MableShop
        </Link>

        <nav className="hidden items-center gap-6 md:flex" aria-label="Main navigation">
          <Link to="/products" className="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">
            Products
          </Link>
          <Link to="/cart" className="relative text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">
            <ShoppingCart className="h-5 w-5" />
            {totalItems > 0 && (
              <Badge className="absolute -right-3 -top-3 h-5 w-5 items-center justify-center rounded-full p-0 text-xs">
                {totalItems}
              </Badge>
            )}
          </Link>
          {user ? (
            <div className="flex items-center gap-3">
              <span className="text-sm text-muted-foreground">{user.name}</span>
              <Button variant="ghost" size="icon" onClick={handleLogout} aria-label="Log out">
                <LogOut className="h-5 w-5" />
              </Button>
            </div>
          ) : (
            <Button asChild variant="default" size="sm">
              <Link to="/login">
                <User className="mr-2 h-4 w-4" />
                Sign In
              </Link>
            </Button>
          )}
        </nav>

        <Button
          variant="ghost"
          size="icon"
          className="md:hidden"
          onClick={() => setMenuOpen(!menuOpen)}
          aria-label={menuOpen ? "Close menu" : "Open menu"}
        >
          {menuOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
        </Button>
      </div>

      {menuOpen && (
        <div className="border-t md:hidden">
          <nav className="flex flex-col gap-2 px-4 py-4" aria-label="Mobile navigation">
            <Link to="/products" className="text-sm font-medium py-2" onClick={() => setMenuOpen(false)}>
              Products
            </Link>
            <Link to="/cart" className="text-sm font-medium py-2" onClick={() => setMenuOpen(false)}>
              Cart {totalItems > 0 && `(${totalItems})`}
            </Link>
            {user ? (
              <>
                <span className="text-sm text-muted-foreground py-2">{user.name}</span>
                <Button variant="ghost" size="sm" onClick={() => { handleLogout(); setMenuOpen(false); }}>
                  <LogOut className="mr-2 h-4 w-4" />
                  Log out
                </Button>
              </>
            ) : (
              <Button asChild size="sm" onClick={() => setMenuOpen(false)}>
                <Link to="/login">Sign In</Link>
              </Button>
            )}
          </nav>
        </div>
      )}
    </header>
  );
}
