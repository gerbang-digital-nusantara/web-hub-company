/* Paket langganan — dibaca dari /assets/data/paket.json (sumber tunggal,
 * juga dibaca server untuk validasi & estimasi). Dipakai beranda (kartu
 * Harga) dan halaman Daftar. */
(function () {
  "use strict";

  let cache = null;

  function load() {
    if (!cache) {
      cache = fetch("assets/data/paket.json", { cache: "no-cache" })
        .then((r) => {
          if (!r.ok) throw new Error(`HTTP ${r.status}`);
          return r.json();
        })
        .catch((e) => {
          cache = null; // boleh dicoba lagi
          throw e;
        });
    }
    return cache;
  }

  const rupiah = (n) => `Rp ${Number(n || 0).toLocaleString("id-ID")}`;
  // 299000 → "299rb", 2990000 → "2,99jt" — untuk angka besar di kartu harga.
  const ringkas = (n) => {
    if (n >= 1e6) return `${(n / 1e6).toLocaleString("id-ID", { maximumFractionDigits: 2 })}jt`;
    if (n >= 1e3) return `${(n / 1e3).toLocaleString("id-ID", { maximumFractionDigits: 0 })}rb`;
    return String(n);
  };

  const penawaran = (p) => !(Number(p?.harga_bulanan) > 0);

  // Harus sama dengan paketFile.tagihan di paket.go.
  function hitungTagihan(data, p, periode) {
    if (penawaran(p)) return 0;
    const bln = Number(p.harga_bulanan);
    return periode === "tahunan" ? bln * (12 - Number(data.diskon_tahunan_bulan || 0)) : bln;
  }

  const esc = (v) =>
    String(v ?? "").replace(/[&<>"']/g, (c) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c],
    );

  // Kartu Harga di beranda. Tombol tiap paket menuju halaman Daftar dengan
  // paket itu terpilih.
  function renderKartu(el, data) {
    const hemat = Number(data.diskon_tahunan_bulan || 0);
    el.innerHTML = data.paket
      .map((p) => {
        const harga = penawaran(p)
          ? `<div class="price"><span class="amt amt-sm">Hubungi kami</span></div>
             <div class="price-note">Penawaran disesuaikan</div>`
          : `<div class="price"><span class="cur">Rp</span><span class="amt">${ringkas(p.harga_bulanan)}</span><span class="per">/bulan</span></div>
             <div class="price-note">${
               hemat > 0
                 ? `atau ${rupiah(hitungTagihan(data, p, "tahunan"))}/tahun — hemat ${hemat} bulan`
                 : "Ditagih bulanan"
             }</div>`;
        const btnKelas = p.rekomendasi ? "btn-primary" : penawaran(p) ? "btn-navy" : "btn-outline";
        const btnLabel = penawaran(p) ? "Ajukan Penawaran" : `Daftar ${esc(p.nama)}`;
        return `
          <article class="plan${p.rekomendasi ? " is-featured" : ""}">
            ${p.rekomendasi ? '<span class="plan-badge">Rekomendasi</span>' : ""}
            <h3>${esc(p.nama)}</h3>
            <div class="plan-for">${esc(p.untuk)}</div>
            ${harga}
            <ul class="checks">
              ${(p.fitur || []).map((f) => `<li><i class="bi bi-check2"></i>${esc(f)}</li>`).join("")}
            </ul>
            <a class="btn ${btnKelas} btn-block" href="daftar.html?paket=${encodeURIComponent(p.id)}">${btnLabel}</a>
          </article>`;
      })
      .join("");
  }

  window.HubPaket = { load, rupiah, ringkas, penawaran, hitungTagihan, esc, renderKartu };
})();
