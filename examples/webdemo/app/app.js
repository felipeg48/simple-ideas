// Northwind Shop: a tiny static shop (no network, no build step). The
// signed-in customer lives in sessionStorage, the orders below.
var ORDERS = {
  4711: { items: [["Harbor blend coffee, 1 kg", 24.5], ["Two mugs", 18.0]], paid: false },
  4712: { items: [["Tea sampler", 12.9]], paid: true },
  4698: { items: [["Grinder", 64.0]], paid: true }
};
var SHIPPING = { Standard: 4.9, Express: 9.9 };
var GIFT = 2.5;

function $(s) { return document.querySelector(s); }
function euro(n) { return "€" + n.toFixed(2); }
function user() { return sessionStorage.getItem("user"); }
function needUser() { if (!user()) location.replace("index.html"); }
function param(k) { return new URLSearchParams(location.search).get(k); }

function signIn() {
  var form = $("#signin");
  form.addEventListener("submit", function (e) {
    e.preventDefault();
    var u = $("#username").value.trim(), p = $("#password").value;
    if (u === "demo" && p === "shop-demo") {
      sessionStorage.setItem("user", u);
      location.href = "orders.html";
    } else {
      $("#signin-error").textContent = "That username and password do not match.";
    }
  });
}

function orders() {
  needUser();
  $("#who").textContent = user();
  var list = $("#orders"), search = $("#search"), showPaid = $("#show-paid");
  function draw() {
    var q = search.value.trim();
    list.innerHTML = "";
    Object.keys(ORDERS).sort().reverse().forEach(function (id) {
      var o = ORDERS[id];
      if (q && id.indexOf(q) < 0) return;
      if (o.paid && !showPaid.checked) return;
      var li = document.createElement("li");
      var total = o.items.reduce(function (s, it) { return s + it[1]; }, 0);
      li.innerHTML = '<a href="order.html?id=' + id + '">Order #' + id + '</a><span class="muted">' +
        euro(total) + " · " + (o.paid ? "paid" : "waiting for payment") + "</span>";
      list.appendChild(li);
    });
    if (!list.children.length) list.innerHTML = '<li class="muted">No orders match.</li>';
  }
  search.addEventListener("input", draw);
  showPaid.addEventListener("change", draw);
  document.addEventListener("keydown", function (e) {
    if (e.key === "/" && document.activeElement !== search) { e.preventDefault(); search.focus(); }
  });
  draw();
}

function order() {
  needUser();
  var id = param("id"), o = ORDERS[id];
  if (!o) { $("main").innerHTML = "<h1>No such order</h1>"; return; }
  $("#title").textContent = "Order #" + id;
  var body = $("#items");
  o.items.forEach(function (it) {
    var tr = document.createElement("tr");
    tr.innerHTML = "<td>" + it[0] + '</td><td class="n">' + euro(it[1]) + "</td>";
    body.appendChild(tr);
  });
  var ship = $("#shipping"), gift = $("#gift"), discount = 0;
  function total() {
    var t = o.items.reduce(function (s, it) { return s + it[1]; }, 0) + SHIPPING[ship.value] + (gift.checked ? GIFT : 0);
    $("#total").textContent = "Total: " + euro(t * (1 - discount));
  }
  ship.addEventListener("change", total);
  gift.addEventListener("change", total);
  total();
  $("#apply").addEventListener("click", function () {
    var code = $("#coupon").value.trim().toUpperCase();
    // coupons are checked by the shop's server; this copy of the shop has none
    fetch("api/coupons/" + encodeURIComponent(code)).then(function (r) {
      if (!r.ok) throw new Error("coupon service answered " + r.status);
      return r.json();
    }).then(function (c) { discount = c.percent / 100; $("#coupon-note").textContent = c.percent + "% off"; total(); })
      .catch(function (err) { console.error(err.message); $("#coupon-note").textContent = "That coupon is not valid."; });
  });
  $("#pay").addEventListener("click", function () {
    $("#payment").hidden = false;
    $("#card").focus();
  });
  var card = $("#card"), confirm = $("#confirm");
  card.addEventListener("input", function () {
    confirm.disabled = card.value.replace(/\D/g, "").length !== 16;
  });
  confirm.addEventListener("click", function () {
    confirm.disabled = true;
    confirm.textContent = "Processing…";
    setTimeout(function () { location.href = "receipt.html?order=" + id; }, 600);
  });
}

function receipt() {
  needUser();
  var id = param("order");
  $("#title").textContent = "Order #" + id;
}
