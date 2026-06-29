(function () {
  "use strict";

  if (window.__tracker) return;

  var API_URL =
    window._TRACKER_API || "http://localhost:8081/track/events";

  var SID =
    sessionStorage.getItem("_tsid") ||
    (function () {
      var id = crypto.randomUUID();
      sessionStorage.setItem("_tsid", id);
      return id;
    })();

  var _prevCart = (function () {
    try { return JSON.parse(localStorage.getItem("cart") || "[]"); } catch (e) { return []; }
  })();

  var _checkoutFired = false;
  var _paymentFired = false;
  var _purchaseFired = false;

  function now() {
    return new Date().toISOString();
  }

  function baseEvent(type, data) {
    return {
      id: crypto.randomUUID(),
      type: type,
      timestamp: now(),
      data: data || {},
      user_agent: navigator.userAgent,
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      location: window.location.hostname,
      url: window.location.href,
      session_id: SID,
    };
  }

  function send(type, data) {
    var body = JSON.stringify([baseEvent(type, data)]);
    if (navigator.sendBeacon) {
      navigator.sendBeacon(API_URL, body);
    } else {
      fetch(API_URL, { method: "POST", body: body, keepalive: true }).catch(
        function () {}
      );
    }
  }

  function diffCart(oldCart, newCart) {
    for (var i = 0; i < newCart.length; i++) {
      var ni = newCart[i];
      if (!ni || !ni.product) continue;
      var oldItem = null;
      for (var j = 0; j < oldCart.length; j++) {
        if (oldCart[j] && oldCart[j].product && oldCart[j].product.id === ni.product.id) {
          oldItem = oldCart[j];
          break;
        }
      }
      if (!oldItem) {
        send("add_to_cart", {
          product_id: ni.product.id,
          product_title: ni.product.title,
          price: ni.product.price,
          quantity: ni.quantity || 1,
        });
      } else if (ni.quantity > oldItem.quantity) {
        send("add_to_cart", {
          product_id: ni.product.id,
          product_title: ni.product.title,
          price: ni.product.price,
          quantity: ni.quantity - oldItem.quantity,
        });
      }
    }
  }

  function waitForElement(selector, timeout, callback) {
    var elapsed = 0;
    var interval = 100;
    var timer = setInterval(function () {
      var el = document.querySelector(selector);
      if (el) {
        clearInterval(timer);
        callback(el);
        return;
      }
      elapsed += interval;
      if (elapsed >= timeout) {
        clearInterval(timer);
      }
    }, interval);
  }

  function waitForText(pattern, timeout, callback) {
    var elapsed = 0;
    var interval = 200;
    var timer = setInterval(function () {
      var all = document.body ? document.body.innerText : "";
      if (pattern.test(all)) {
        clearInterval(timer);
        callback();
        return;
      }
      elapsed += interval;
      if (elapsed >= timeout) {
        clearInterval(timer);
      }
    }, interval);
  }

  function detectPaymentStep() {
    if (_paymentFired) return;
    _paymentFired = true;
    send("payment_info", { method: "card" });
  }

  function detectPurchase(cartSnapshot) {
    if (_purchaseFired) return;
    _purchaseFired = true;
    var orderNum = "";
    var all = document.body ? document.body.innerText : "";
    var match = all.match(/ORD-[A-Z0-9]+/);
    if (match) orderNum = match[0];
    send("purchase", {
      order_id: orderNum || "unknown",
      items: (cartSnapshot || []).map(function (i) {
        return {
          product_id: i.product ? i.product.id : null,
          title: i.product ? i.product.title : "",
          price: i.product ? i.product.price : 0,
          quantity: i.quantity || 0,
        };
      }),
      total: (cartSnapshot || []).reduce(function (s, i) {
        return s + (i.product ? i.product.price : 0) * (i.quantity || 0);
      }, 0),
    });
  }

  function onCheckoutPage() {
    if (_checkoutFired) return;
    _checkoutFired = true;
    send("checkout", {
      items: _prevCart.map(function (i) {
        return {
          product_id: i.product ? i.product.id : null,
          title: i.product ? i.product.title : "",
          price: i.product ? i.product.price : 0,
          quantity: i.quantity || 0,
        };
      }),
      total: _prevCart.reduce(function (s, i) {
        return s + (i.product ? i.product.price : 0) * (i.quantity || 0);
      }, 0),
    });
    waitForElement(
      'input[placeholder*="4242"], input[placeholder*="MM/YY"], input[placeholder*="CVV"], [id*="checkout-card"], [id*="checkout-expiry"]',
      15000,
      detectPaymentStep
    );
  }

  function onCartChanged(newCart) {
    diffCart(_prevCart, newCart);
    var wasNonEmpty = _prevCart.length > 0;
    var nowEmpty = newCart.length === 0;
    if (wasNonEmpty && nowEmpty && _checkoutFired && !_purchaseFired) {
      var snapshot = _prevCart;
      waitForText(/order\s*confirmed|ORD-[A-Z0-9]+/i, 5000, function () {
        detectPurchase(snapshot);
      });
    }
    _prevCart = newCart;
  }

  function onUrlChange() {
    var path = window.location.pathname;
    if (path === "/checkout" || path.endsWith("/checkout")) {
      onCheckoutPage();
    }
  }

  function initCartWatch() {
    var originalSetItem = Storage.prototype.setItem;
    Storage.prototype.setItem = function (key, value) {
      if (key === "cart") {
        try {
          var newCart = JSON.parse(value);
          onCartChanged(newCart);
        } catch (e) {}
      }
      return originalSetItem.call(this, key, value);
    };
    try {
      var current = JSON.parse(localStorage.getItem("cart") || "[]");
      if (current.length !== _prevCart.length) _prevCart = current;
    } catch (e) {}
  }

  function initClickTracking() {
    document.addEventListener("click", function (e) {
      var target = e.target.closest(
        "a, button, [role=button], [data-track-click]"
      );
      if (target) {
        send("click", {
          tag: target.tagName,
          id: target.id || "",
          class: target.className || "",
          text: (target.textContent || "").trim().slice(0, 200),
          href: target.href || "",
        });
      }
    });
  }

  function initPageViewTracking() {
    if (document.readyState === "complete") {
      send("page_view", { path: window.location.pathname, referrer: document.referrer || "" });
    } else {
      window.addEventListener("load", function () {
        send("page_view", { path: window.location.pathname, referrer: document.referrer || "" });
      });
    }
    var originalPushState = history.pushState;
    var originalReplaceState = history.replaceState;
    history.pushState = function () {
      originalPushState.apply(this, arguments);
      setTimeout(onUrlChange, 50);
      send("page_view", { path: window.location.pathname, referrer: "" });
    };
    history.replaceState = function () {
      originalReplaceState.apply(this, arguments);
      setTimeout(onUrlChange, 50);
      send("page_view", { path: window.location.pathname, referrer: "" });
    };
    window.addEventListener("popstate", function () {
      setTimeout(onUrlChange, 50);
      send("page_view", { path: window.location.pathname, referrer: "" });
    });
  }

  function initLeadTracking() {
    document.addEventListener("submit", function (e) {
      var form = e.target;
      if (!form) return;
      var emailInput = form.querySelector('input[type="email"]');
      var nameInput = form.querySelector('input[autocomplete="name"]');
      if (emailInput && emailInput.value && nameInput) {
        setTimeout(function () {
          send("lead", { email: emailInput.value });
        }, 100);
      }
    });
  }

  window.__tracker = {
    send: send,
    pageView: function () {
      send("page_view", { path: window.location.pathname, referrer: "" });
    },
  };

  initCartWatch();
  initClickTracking();
  initPageViewTracking();
  initLeadTracking();

  setTimeout(onUrlChange, 100);
})();
