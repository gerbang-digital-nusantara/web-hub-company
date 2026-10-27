/* Paket langganan — katalog hub-control (menu Paket & Harga), dibaca server
 * dan disajikan di /api/paket (sama dengan yang dipakai validasi & estimasi
 * di daftar.go). Pratinjau GitHub Pages (tanpa server) memakai cadangan
 * assets/data/paket.json. Dipakai beranda (kartu Harga) dan halaman Daftar. */
(function () {
  "use strict";

  let cache = null;

  // paket.json (format lama) → bentuk /api/paket: harga_tahunan & penawaran.
  function normalisasi(d) {
    const diskon = Number(d.diskon_tahunan_bulan || 0);
    d.paket.forEach((p) => {
      if (p.harga_tahunan == null) p.harga_tahunan = Number(p.harga_bulanan || 0) * (12 - diskon);
      if (p.penawaran == null) p.penawaran = !(Number(p.harga_bulanan) > 0);
    });
    return d;
  }

  function load() {
    if (!cache) {
      const url = window.HUB_PREVIEW ? "assets/data/paket.json" : "api/paket";
      cache = fetch(url, { cache: "no-cache" })
        .then((r) => {
          if (!r.ok) throw new Error(`HTTP ${r.status}`);
          return r.json();
        })
        .then((j) => {
          const d = j && Array.isArray(j.paket) ? j : j && j.data;
          if (!d || !Array.isArray(d.paket) || !d.paket.length) throw new Error("katalog kosong");
          return normalisasi(d);
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

  const penawaran = (p) => !!p?.penawaran;
  // 0 = paket tidak dijual per periode itu.
  const tersedia = (p, periode) =>
    Number(periode === "tahunan" ? p?.harga_tahunan : p?.harga_bulanan) > 0;

  // Harus sama dengan paketFile.tagihan di paket.go. `data` tidak dipakai
  // lagi (dulu diskon tahunan global), dipertahankan supaya pemanggil tetap.
  function hitungTagihan(data, p, periode) {
    if (penawaran(p)) return 0;
    return Number(periode === "tahunan" ? p.harga_tahunan : p.harga_bulanan) || 0;
  }

  // Hemat bayar tahunan dibanding 12× bulanan: "2 bulan" bila pas kelipatan
  // harga bulanan, selain itu nominal rupiah. "" bila tidak hemat.
  function hemat(p) {
    const bln = Number(p.harga_bulanan) || 0;
    const sel = bln * 12 - (Number(p.harga_tahunan) || 0);
    if (!bln || !p.harga_tahunan || sel <= 0) return "";
    return sel % bln === 0 ? `${sel / bln} bulan` : rupiah(sel);
  }

  const esc = (v) =>
    String(v ?? "").replace(/[&<>"']/g, (c) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c],
    );

  // Kartu Harga di beranda. Tombol tiap paket menuju halaman Daftar dengan
  // paket itu terpilih.
  function renderKartu(el, data) {
    el.innerHTML = data.paket
      .map((p) => {
        const h = hemat(p);
        const bulanan = tersedia(p, "bulanan");
        const harga = penawaran(p)
          ? `<div class="price"><span class="amt amt-sm">Hubungi kami</span></div>
             <div class="price-note">Penawaran disesuaikan</div>`
          : `<div class="price"><span class="cur">Rp</span><span class="amt">${ringkas(
              bulanan ? p.harga_bulanan : p.harga_tahunan,
            )}</span><span class="per">/${bulanan ? "bulan" : "tahun"}</span></div>
             <div class="price-note">${
               !bulanan
                 ? "Ditagih tahunan"
                 : tersedia(p, "tahunan")
                   ? `atau ${rupiah(p.harga_tahunan)}/tahun${h ? ` — hemat ${h}` : ""}`
                   : "Ditagih bulanan"
             }</div>`;
        const btnKelas = p.rekomendasi ? "btn-primary" : penawaran(p) ? "btn-dark" : "btn-outline";
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

  window.HubPaket = { load, rupiah, ringkas, penawaran, tersedia, hitungTagihan, hemat, esc, renderKartu };
})();
