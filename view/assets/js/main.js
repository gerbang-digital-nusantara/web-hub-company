(function () {
  "use strict";

  document.documentElement.classList.add("js");

  const $ = (sel, root = document) => root.querySelector(sel);
  const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));

  /* ===================== Header & menu ponsel ===================== */

  const header = $("#header");
  const onScroll = () => header?.classList.toggle("is-scrolled", window.scrollY > 8);
  window.addEventListener("scroll", onScroll, { passive: true });
  onScroll();

  const toggle = $(".nav-toggle");
  const menu = $("#navMenu");
  const setMenu = (open) => {
    menu?.classList.toggle("is-open", open);
    toggle?.setAttribute("aria-expanded", String(open));
    toggle?.setAttribute("aria-label", open ? "Tutup menu" : "Buka menu");
    const ic = toggle?.querySelector("i");
    if (ic) ic.className = open ? "bi bi-x-lg" : "bi bi-list";
  };
  toggle?.addEventListener("click", () => setMenu(!menu.classList.contains("is-open")));
  // Pilih tautan di menu ponsel → menu ditutup.
  menu?.addEventListener("click", (e) => {
    if (e.target.closest("a")) setMenu(false);
  });
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") setMenu(false);
  });

  /* ===================== Kontak (config.js) ===================== */

  const cfg = window.HUB_CONTACT || {};
  const wa = String(cfg.whatsapp || "").replace(/[^0-9]/g, "");
  const email = String(cfg.email || "").trim();

  if (wa) {
    const url = `https://wa.me/${wa}${cfg.waText ? `?text=${encodeURIComponent(cfg.waText)}` : ""}`;
    // Tampilan nomor: 6281234567890 → +62 812-3456-7890
    const label = wa.startsWith("62")
      ? `+62 ${wa.slice(2).replace(/(\d{3})(\d{4})(\d+)/, "$1-$2-$3")}`
      : wa;
    for (const id of ["linkWa", "waFloat"]) {
      const a = document.getElementById(id);
      if (a) {
        a.href = url;
        a.hidden = false;
      }
    }
    $$('[data-contact="waLabel"]').forEach((el) => (el.textContent = label));
  }
  if (email) {
    const a = $("#linkEmail");
    if (a) {
      a.href = `mailto:${email}`;
      a.hidden = false;
    }
    $$('[data-contact="email"]').forEach((el) => (el.textContent = email));
    $$('[data-contact-href="email"]').forEach((el) => {
      el.href = `mailto:${email}`;
      el.hidden = false;
    });
  }

  const year = $("#year");
  if (year) year.textContent = String(new Date().getFullYear());

  // Spanduk mode pratinjau (GitHub Pages) — lihat HUB_PREVIEW di config.js.
  if (window.HUB_PREVIEW) {
    const bar = document.createElement("div");
    bar.className = "preview-bar";
    bar.innerHTML =
      '<i class="bi bi-eye"></i> Versi pratinjau — form Demo &amp; Daftar bisa dicoba, tetapi tidak dikirim ke mana pun.';
    document.body.prepend(bar);
  }

  /* ===================== Muncul saat digulir ===================== */

  const reveals = $$(".reveal");
  const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  if (reduce || !("IntersectionObserver" in window)) {
    reveals.forEach((el) => el.classList.add("is-in"));
  } else {
    const io = new IntersectionObserver(
      (entries) => {
        entries.forEach((en) => {
          if (!en.isIntersecting) return;
          en.target.classList.add("is-in");
          io.unobserve(en.target);
        });
      },
      { rootMargin: "0px 0px -8% 0px", threshold: 0.08 },
    );
    reveals.forEach((el) => io.observe(el));
  }

  /* ===================== Paket (paket.json) ===================== */

  // Kartu Harga + pilihan paket di form demo, keduanya dari paket.json.
  const cards = $("#pricingCards");
  const pkg = $("#fPackage");
  if ((cards || pkg) && window.HubPaket) {
    window.HubPaket.load()
      .then((data) => {
        if (cards) window.HubPaket.renderKartu(cards, data);
        if (pkg) {
          const awal = pkg.querySelector('option[value=""]');
          pkg.innerHTML = "";
          if (awal) pkg.appendChild(awal);
          data.paket.forEach((p) => pkg.add(new Option(p.nama, p.nama)));
        }
      })
      .catch((e) => {
        console.error("Gagal memuat paket:", e);
        if (cards)
          cards.innerHTML =
            '<p class="pricing-loading">Paket belum bisa dimuat. <a href="#demo">Hubungi kami</a> untuk info harga.</p>';
      });
  }

  /* ===================== Form demo ===================== */

  const form = $("#demoForm");
  if (!form) return; // halaman tanpa form demo (mis. daftar.html)
  const submit = $("#demoSubmit");
  const status = $("#demoStatus");

  const clearErrors = () => {
    $$(".field", form).forEach((f) => f.classList.remove("has-error"));
    $$("[data-err]", form).forEach((el) => (el.textContent = ""));
  };
  const showErrors = (errs) => {
    let first = null;
    Object.entries(errs || {}).forEach(([key, msg]) => {
      const el = $(`[data-err="${key}"]`, form);
      if (!el) return;
      el.textContent = msg;
      el.closest(".field")?.classList.add("has-error");
      first = first || el.closest(".field")?.querySelector("input,select,textarea");
    });
    first?.focus();
  };
  const setStatus = (kind, msg) => {
    status.className = `form-status ${kind}`;
    status.textContent = msg;
  };

  // Validasi sisi klien = cermin aturan demo.go (server tetap memeriksa ulang).
  // Nama usaha & email wajib karena pendaftar diteruskan ke hub-control
  // sebagai tenant coba gratis.
  const reSub = /^[a-z0-9][a-z0-9-]{1,28}[a-z0-9]$/;
  const validate = (d) => {
    const e = {};
    if (!d.name) e.name = "Nama wajib diisi";
    if (!d.company) e.company = "Nama usaha wajib diisi";
    if (!d.phone && !d.email) e.phone = "Isi nomor WhatsApp atau email";
    if (d.phone && !/^[0-9+\-\s()]{8,20}$/.test(d.phone)) e.phone = "Nomor WhatsApp tidak valid";
    if (!d.email) e.email = "Email wajib diisi";
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(d.email)) e.email = "Email tidak valid";
    if (d.subdomain && !reSub.test(d.subdomain)) e.subdomain = "3–30 karakter: huruf kecil, angka, atau tanda hubung";
    return e;
  };

  /* Cek ketersediaan alamat {subdomain}.hub.xpanca.com sambil mengetik
   * (GET api/demo/subdomain → hub-control). Hanya petunjuk, bukan reservasi. */
  const sub = $("#fSub");
  const subHint = $("#subHint");
  const subHintAwal = subHint?.textContent || "";
  const setSubHint = (kind, msg) => {
    if (!subHint) return;
    subHint.className = `hint${kind ? ` ${kind}` : ""}`;
    subHint.textContent = msg;
  };
  let subTimer = 0;
  let subSeq = 0;
  sub?.addEventListener("input", () => {
    // Ketik bebas → paksa huruf kecil, buang karakter yang tak diizinkan.
    const v = sub.value.toLowerCase().replace(/[^a-z0-9-]/g, "");
    if (v !== sub.value) sub.value = v;
    clearTimeout(subTimer);
    const seq = ++subSeq;
    if (!v) return setSubHint("", subHintAwal);
    if (!reSub.test(v)) return setSubHint("bad", "3–30 karakter: huruf kecil, angka, atau tanda hubung");
    if (window.HUB_PREVIEW) return setSubHint("", `Alamat: ${v}.hub.xpanca.com`);
    setSubHint("", "Memeriksa…");
    subTimer = setTimeout(async () => {
      try {
        const res = await fetch(`api/demo/subdomain?name=${encodeURIComponent(v)}`);
        const json = await res.json().catch(() => ({}));
        if (seq !== subSeq) return; // sudah diketik ulang
        const d = json.data;
        if (!res.ok || !d) return setSubHint("", subHintAwal);
        if (d.available) setSubHint("ok", `✓ ${d.domain || `${v}.hub.xpanca.com`} tersedia`);
        else setSubHint("bad", d.reason || "Alamat ini sudah dipakai, coba yang lain");
      } catch (e) {
        if (seq === subSeq) setSubHint("", subHintAwal);
      }
    }, 450);
  });

  form.addEventListener("submit", async (ev) => {
    ev.preventDefault();
    clearErrors();
    status.className = "form-status";

    const data = Object.fromEntries(
      Array.from(new FormData(form).entries()).map(([k, v]) => [k, String(v).trim()]),
    );
    const errs = validate(data);
    if (Object.keys(errs).length) {
      showErrors(errs);
      return;
    }

    if (window.HUB_PREVIEW) {
      // Tanpa server: tampilkan hasil seolah terkirim, tanpa menyimpan apa pun.
      form.reset();
      setStatus("ok", "Pratinjau: form valid. Di situs asli, permintaan ini terkirim ke tim kami.");
      return;
    }

    const label = submit.innerHTML;
    submit.disabled = true;
    submit.innerHTML = '<i class="bi bi-hourglass-split"></i> Mengirim…';
    try {
      const res = await fetch("api/demo", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      });
      const json = await res.json().catch(() => ({}));
      if (res.ok && json.status) {
        form.reset();
        setSubHint("", subHintAwal);
        // Bila tercatat di hub-control, tampilkan alamat aplikasinya.
        const dom = json.data?.domain || (json.data?.slug ? `${json.data.slug}.hub.xpanca.com` : "");
        setStatus(
          "ok",
          (json.message || "Terima kasih, tim kami akan segera menghubungi Anda.") +
            (dom ? ` Alamat aplikasi Anda: ${dom}` : ""),
        );
      } else {
        if (json.errors) showErrors(json.errors);
        setStatus("err", json.message || "Gagal mengirim. Silakan coba lagi.");
      }
    } catch (e) {
      setStatus(
        "err",
        wa
          ? "Koneksi bermasalah. Silakan coba lagi, atau hubungi kami lewat WhatsApp."
          : "Koneksi bermasalah. Silakan coba lagi.",
      );
    } finally {
      submit.disabled = false;
      submit.innerHTML = label;
    }
  });
})();
