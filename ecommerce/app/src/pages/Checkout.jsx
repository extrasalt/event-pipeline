import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Badge } from "@/components/ui/badge";
import { useCartStore } from "@/stores/cartStore";
import { Loader2, Check, ChevronLeft, ChevronRight, CreditCard, Truck, ClipboardList, ShoppingBag } from "lucide-react";

const STEPS = [
  { label: "Shipping", icon: Truck },
  { label: "Payment", icon: CreditCard },
  { label: "Review", icon: ClipboardList },
  { label: "Confirmation", icon: ShoppingBag },
];

function StepIndicator({ current }) {
  return (
    <nav aria-label="Checkout progress" className="mb-8">
      <ol className="flex items-center justify-center gap-2 sm:gap-4">
        {STEPS.map((step, idx) => {
          const isActive = idx === current;
          const isDone = idx < current;
          const Icon = step.icon;
          return (
            <li key={step.label} className="flex items-center gap-2">
              <div
                className={`flex h-8 w-8 items-center justify-center rounded-full text-xs font-semibold ${
                  isDone ? "bg-primary text-primary-foreground" :
                  isActive ? "bg-primary text-primary-foreground ring-2 ring-primary ring-offset-2" :
                  "bg-muted text-muted-foreground"
                }`}
                aria-current={isActive ? "step" : undefined}
              >
                {isDone ? <Check className="h-4 w-4" /> : Icon && <Icon className="h-4 w-4" />}
              </div>
              <span className={`hidden text-sm font-medium sm:inline ${isActive ? "text-foreground" : "text-muted-foreground"}`}>
                {step.label}
              </span>
              {idx < STEPS.length - 1 && <Separator className="hidden w-8 sm:inline-block" />}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}

function ShippingStep({ data, onChange, onNext }) {
  const [errs, setErrs] = useState({});

  const validate = () => {
    const e = {};
    if (!data.name.trim()) e.name = "Name is required";
    if (!data.address.trim()) e.address = "Address is required";
    if (!data.city.trim()) e.city = "City is required";
    if (!data.state.trim()) e.state = "State is required";
    if (!data.zip.trim()) e.zip = "ZIP code is required";
    if (!data.country.trim()) e.country = "Country is required";
    setErrs(e);
    return Object.keys(e).length === 0;
  };

  const handle = (e) => {
    e.preventDefault();
    if (validate()) onNext();
  };

  const set = (field) => (e) => onChange({ ...data, [field]: e.target.value });

  return (
    <form onSubmit={handle} noValidate>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <Truck className="h-5 w-5" /> Shipping Address
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="checkout-name">Full name</Label>
            <Input id="checkout-name" value={data.name} onChange={set("name")} aria-invalid={!!errs.name} aria-describedby={errs.name ? "err-name" : undefined} />
            {errs.name && <p id="err-name" className="text-xs text-destructive">{errs.name}</p>}
          </div>
          <div className="space-y-2">
            <Label htmlFor="checkout-address">Address</Label>
            <Input id="checkout-address" value={data.address} onChange={set("address")} aria-invalid={!!errs.address} aria-describedby={errs.address ? "err-address" : undefined} />
            {errs.address && <p id="err-address" className="text-xs text-destructive">{errs.address}</p>}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="checkout-city">City</Label>
              <Input id="checkout-city" value={data.city} onChange={set("city")} aria-invalid={!!errs.city} aria-describedby={errs.city ? "err-city" : undefined} />
              {errs.city && <p id="err-city" className="text-xs text-destructive">{errs.city}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="checkout-state">State</Label>
              <Input id="checkout-state" value={data.state} onChange={set("state")} aria-invalid={!!errs.state} aria-describedby={errs.state ? "err-state" : undefined} />
              {errs.state && <p id="err-state" className="text-xs text-destructive">{errs.state}</p>}
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="checkout-zip">ZIP code</Label>
              <Input id="checkout-zip" value={data.zip} onChange={set("zip")} aria-invalid={!!errs.zip} aria-describedby={errs.zip ? "err-zip" : undefined} />
              {errs.zip && <p id="err-zip" className="text-xs text-destructive">{errs.zip}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="checkout-country">Country</Label>
              <Input id="checkout-country" value={data.country} onChange={set("country")} aria-invalid={!!errs.country} aria-describedby={errs.country ? "err-country" : undefined} />
              {errs.country && <p id="err-country" className="text-xs text-destructive">{errs.country}</p>}
            </div>
          </div>
        </CardContent>
        <CardFooter className="justify-end">
          <Button type="submit">
            Continue to Payment <ChevronRight className="ml-2 h-4 w-4" />
          </Button>
        </CardFooter>
      </Card>
    </form>
  );
}

function PaymentStep({ data, onChange, onBack, onNext }) {
  const [errs, setErrs] = useState({});

  const validate = () => {
    const e = {};
    if (!data.cardNumber.trim()) e.cardNumber = "Card number is required";
    else if (!/^\d{16}$/.test(data.cardNumber.replace(/\s/g, ""))) e.cardNumber = "Enter a valid 16-digit card number";
    if (!data.expiry.trim()) e.expiry = "Expiry is required";
    else if (!/^\d{2}\/\d{2}$/.test(data.expiry)) e.expiry = "Use MM/YY format";
    if (!data.cvv.trim()) e.cvv = "CVV is required";
    else if (!/^\d{3,4}$/.test(data.cvv)) e.cvv = "Invalid CVV";
    setErrs(e);
    return Object.keys(e).length === 0;
  };

  const handle = (e) => {
    e.preventDefault();
    if (validate()) onNext();
  };

  const set = (field) => (e) => onChange({ ...data, [field]: e.target.value });

  return (
    <form onSubmit={handle} noValidate>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <CreditCard className="h-5 w-5" /> Payment Information
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="rounded-md bg-muted p-3 text-sm text-muted-foreground">
            This is a simulated checkout. No real payment will be processed.
          </div>
          <div className="space-y-2">
            <Label htmlFor="checkout-card">Card number</Label>
            <Input id="checkout-card" placeholder="4242 4242 4242 4242" value={data.cardNumber} onChange={set("cardNumber")} aria-invalid={!!errs.cardNumber} aria-describedby={errs.cardNumber ? "err-card" : undefined} />
            {errs.cardNumber && <p id="err-card" className="text-xs text-destructive">{errs.cardNumber}</p>}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="checkout-expiry">Expiry (MM/YY)</Label>
              <Input id="checkout-expiry" placeholder="12/28" value={data.expiry} onChange={set("expiry")} aria-invalid={!!errs.expiry} aria-describedby={errs.expiry ? "err-expiry" : undefined} />
              {errs.expiry && <p id="err-expiry" className="text-xs text-destructive">{errs.expiry}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="checkout-cvv">CVV</Label>
              <Input id="checkout-cvv" placeholder="123" type="password" value={data.cvv} onChange={set("cvv")} aria-invalid={!!errs.cvv} aria-describedby={errs.cvv ? "err-cvv" : undefined} />
              {errs.cvv && <p id="err-cvv" className="text-xs text-destructive">{errs.cvv}</p>}
            </div>
          </div>
        </CardContent>
        <CardFooter className="justify-between">
          <Button type="button" variant="outline" onClick={onBack}>
            <ChevronLeft className="mr-2 h-4 w-4" /> Back
          </Button>
          <Button type="submit">
            Review Order <ChevronRight className="ml-2 h-4 w-4" />
          </Button>
        </CardFooter>
      </Card>
    </form>
  );
}

function ReviewStep({ shipping, payment, cartItems, subtotal, onBack, onConfirm, isProcessing }) {
  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <Truck className="h-5 w-5" /> Shipping
          </CardTitle>
        </CardHeader>
        <CardContent className="text-sm">
          <p className="font-medium">{shipping.name}</p>
          <p>{shipping.address}</p>
          <p>{shipping.city}, {shipping.state} {shipping.zip}</p>
          <p>{shipping.country}</p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <CreditCard className="h-5 w-5" /> Payment
          </CardTitle>
        </CardHeader>
        <CardContent className="text-sm">
          <p>Card ending in {payment.cardNumber.replace(/\s/g, "").slice(-4)}</p>
          <p>Expires {payment.expiry}</p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <ShoppingBag className="h-5 w-5" /> Items
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {cartItems.map(({ product, quantity }) => (
            <div key={product.id} className="flex items-center justify-between text-sm">
              <div className="flex items-center gap-3">
                <div className="h-10 w-10 flex-shrink-0 overflow-hidden rounded bg-muted">
                  <img src={product.image} alt={product.title} className="h-full w-full object-cover" />
                </div>
                <div>
                  <p className="font-medium">{product.title}</p>
                  <p className="text-muted-foreground">Qty: {quantity}</p>
                </div>
              </div>
              <span className="font-medium">${(product.price * quantity).toFixed(2)}</span>
            </div>
          ))}
          <Separator />
          <div className="flex justify-between font-semibold">
            <span>Total</span>
            <span>${subtotal.toFixed(2)}</span>
          </div>
        </CardContent>
        <CardFooter className="justify-between">
          <Button type="button" variant="outline" onClick={onBack} disabled={isProcessing}>
            <ChevronLeft className="mr-2 h-4 w-4" /> Back
          </Button>
          <Button onClick={onConfirm} disabled={isProcessing}>
            {isProcessing ? (
              <><Loader2 className="mr-2 h-4 w-4 animate-spin" /> Processing...</>
            ) : (
              <>Place Order - ${subtotal.toFixed(2)}</>
            )}
          </Button>
        </CardFooter>
      </Card>
    </div>
  );
}

function ConfirmationStep({ orderNumber, onContinue }) {
  return (
    <Card>
      <CardHeader className="text-center">
        <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-primary/10">
          <Check className="h-8 w-8 text-primary" />
        </div>
        <CardTitle className="text-2xl">Order Confirmed!</CardTitle>
      </CardHeader>
      <CardContent className="text-center">
        <p className="text-muted-foreground">
          Thank you for your purchase. Your order number is:
        </p>
        <p className="mt-2 text-lg font-bold text-primary">{orderNumber}</p>
        <p className="mt-2 text-sm text-muted-foreground">
          A confirmation email will be sent shortly.
        </p>
      </CardContent>
      <CardFooter className="justify-center">
        <Button onClick={onContinue}>
          <ShoppingBag className="mr-2 h-4 w-4" /> Continue Shopping
        </Button>
      </CardFooter>
    </Card>
  );
}

export default function Checkout() {
  const navigate = useNavigate();
  const { items, clearCart, subtotal } = useCartStore();
  const [step, setStep] = useState(0);
  const [isProcessing, setIsProcessing] = useState(false);
  const [orderNumber, setOrderNumber] = useState("");
  const [shipping, setShipping] = useState({ name: "", address: "", city: "", state: "", zip: "", country: "" });
  const [payment, setPayment] = useState({ cardNumber: "", expiry: "", cvv: "" });

  if (items.length === 0 && step < 3) {
    navigate("/cart");
    return null;
  }

  const handleConfirm = async () => {
    setIsProcessing(true);
    await new Promise((r) => setTimeout(r, 2000));
    const num = "ORD-" + Date.now().toString(36).toUpperCase();
    setOrderNumber(num);
    clearCart();
    setIsProcessing(false);
    setStep(3);
  };

  return (
    <main className="mx-auto max-w-2xl px-4 py-8 sm:px-6 lg:px-8">
      <h1 className="mb-8 text-3xl font-bold">Checkout</h1>
      <StepIndicator current={step} />

      {step === 0 && <ShippingStep data={shipping} onChange={setShipping} onNext={() => setStep(1)} />}
      {step === 1 && <PaymentStep data={payment} onChange={setPayment} onBack={() => setStep(0)} onNext={() => setStep(2)} />}
      {step === 2 && (
        <ReviewStep
          shipping={shipping}
          payment={payment}
          cartItems={items}
          subtotal={subtotal}
          onBack={() => setStep(1)}
          onConfirm={handleConfirm}
          isProcessing={isProcessing}
        />
      )}
      {step === 3 && <ConfirmationStep orderNumber={orderNumber} onContinue={() => navigate("/products")} />}
    </main>
  );
}
