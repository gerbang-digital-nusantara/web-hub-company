/* Halaman Daftar Berlangganan (daftar.html).
 *
 * Paket & harga dari paket.json lewat HubPaket (paket.js). Ringkasan di
 * samping hanya tampilan — server menghitung ulang estimasi dari paket.json
 * yang sama dan itulah yang disimpan (daftar.go). */
(function () {
  "use strict";

  const P = window.HubPaket;
  const $ = (sel, root = document) => root.querySelector(sel);
  const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));

  const form = $("#daftarForm");
  if (!form || !P) return;

  const pick = $("#planPick");
  const submit = $("#daftarSubmit");
  const status = $("#daftarStatus");
  let data = null;

  const paketTerpilih = () => {
    const id = form.querySelector('input[name="paket"]:checked')?.value;
    return data?.paket.find((p) => p.id === id) || null;
  };
  const periodeTerpilih = () => form.querySelector('input[name="periode"]:checked')?.value || "";

  /* ===================== Pilihan paket ===================== */

  function renderPilihan() {
    const minta = new URLSearchParams(location.search).get("paket");
    const awal =
      data.paket.find((p) => p.id === minta) || data.paket.find((p) => p.rekomendasi) || data.paket[0];
    pick.innerHTML = data.paket
      .map(
        (p) => `
        <label class="plan-opt">
          <input type="radio" name="paket" value="${P.esc(p.id)}" ${p.id === awal.id ? "checked" : ""} />
          <span class="plan-opt-body">
            <span class="plan-opt-name">${P.esc(p.nama)}${
              p.rekomendasi ? '<span class="plan-opt-tag">Rekomendasi</span>' : ""
            }</span>
            <span class="plan-opt-price">${
              P.penawaran(p) ? "Harga penawaran" : `${P.rupiah(p.harga_bulanan)}/bulan`
            }</span>
            <span class="plan-opt-for">${P.esc(p.untuk)}</span>
          </span>
        </label>`,
      )
      .join("");

    const hemat = Number(data.diskon_tahunan_bulan || 0);
    const badge = $("#hematBadge");
    if (badge && hemat > 0) {
      badge.textContent = `hemat ${hemat} bln`;
      badge.hidden = false;
    }
  }

  /* ===================== Ringkasan ===================== */

  function renderRingkasan() {
    const p = paketTerpilih();
    if (!p) return;
    const tawar = P.penawaran(p);
    const periode = periodeTerpilih();

    $("#periodeWrap").hidden = tawar;
    $("#penawaranInfo").hidden = !tawar;
    $("#sumPeriodeRow").hidden = tawar;

    $("#sumPaket").textContent = p.nama;
    $("#sumPeriode").textContent = periode === "tahunan" ? "Tahunan" : "Bulanan";

    const info = $("#periodeInfo");
    if (tawar) {
      $("#sumLabel").textContent = "Harga";
      $("#sumTotal").textContent = "Penawaran";
      $("#sumNote").textContent = "Tim kami menghitung harga sesuai jumlah gudang, pengguna, dan kebutuhan integrasi Anda.";
    } else {
      const total = P.hitungTagihan(data, p, periode);
      $("#sumLabel").textContent = periode === "tahunan" ? "Tagihan per tahun" : "Tagihan per bulan";
      $("#sumTotal").textContent = P.rupiah(total);
      $("#sumNote").textContent = "Belum termasuk PPN. Tagihan dikirim setelah pendaftaran diverifikasi.";
      if (info) {
        const hemat = Number(data.diskon_tahunan_bulan || 0);
        info.textContent =
          periode === "tahunan"
            ? `${P.rupiah(total)} per tahun — setara ${P.rupiah(Math.round(total / 12))}/bulan${
                hemat > 0 ? `, hemat ${P.rupiah(p.harga_bulanan * hemat)}` : ""
              }.`
            : `${P.rupiah(total)} ditagih setiap bulan.`;
      }
    }
    $("#sumFitur").innerHTML = (p.fitur || [])
      .map((f) => `<li><i class="bi bi-check2"></i>${P.esc(f)}</li>`)
      .join("");
  }

  form.addEventListener("change", (e) => {
    if (e.target.name === "paket" || e.target.name === "periode") {
      clearError(e.target.name);
      renderRingkasan();
    }
  });

  P.load()
    .then((d) => {
      data = d;
      renderPilihan();
      renderRingkasan();
    })
    .catch((e) => {
      console.error("Gagal memuat paket:", e);
      pick.innerHTML =
        '<p class="field-error">Paket belum bisa dimuat. Muat ulang halaman, atau <a href="/#demo">hubungi kami</a>.</p>';
      submit.disabled = true;
    });

  /* ===================== Validasi ===================== */

  function clearError(key) {
    const el = $(`[data-err="${key}"]`, form);
    if (!el) return;
    el.textContent = "";
    el.closest(".field")?.classList.remove("has-error");
  }
  function clearErrors() {
    $$(".field", form).forEach((f) => f.classList.remove("has-error"));
    $$("[data-err]", form).forEach((el) => (el.textContent = ""));
  }
  function showErrors(errs) {
    let first = null;
    Object.entries(errs || {}).forEach(([key, msg]) => {
      const el = $(`[data-err="${key}"]`, form);
      if (!el) return;
      el.textContent = msg;
      const field = el.closest(".field");
      field?.classList.add("has-error");
      if (!first) first = field?.querySelector("input,select,textarea") || form.querySelector(`[name="${key}"]`);
    });
    if (first) {
      first.focus({ preventScroll: true });
      first.scrollIntoView({ block: "center", behavior: "smooth" });
    }
  }
  // Hapus pesan galat sebuah isian begitu diketik ulang.
  form.addEventListener("input", (e) => {
    if (e.target.name) clearError(e.target.name);
  });

  // Cermin aturan daftar.go; server tetap pemutus akhir.
  function validate(d) {
    const e = {};
    const p = data?.paket.find((x) => x.id === d.paket);
    if (!p) e.paket = "Pilih paket";
    else if (!P.penawaran(p) && !["bulanan", "tahunan"].includes(d.periode)) e.periode = "Pilih periode pembayaran";
    if (!d.nama_usaha) e.nama_usaha = "Nama usaha wajib diisi";
    if (d.jumlah_gudang < 0 || d.jumlah_gudang > 10000) e.jumlah_gudang = "Jumlah gudang tidak valid";
    if (d.jumlah_pengguna < 0 || d.jumlah_pengguna > 10000) e.jumlah_pengguna = "Jumlah pengguna tidak valid";
    if (!d.kota) e.kota = "Kota wajib diisi";
    if (d.npwp && ![15, 16].includes(d.npwp.length)) e.npwp = "NPWP harus 15 atau 16 digit";
    if (d.subdomain && !/^[a-z0-9][a-z0-9-]{1,28}[a-z0-9]$/.test(d.subdomain))
      e.subdomain = "3–30 karakter: huruf kecil, angka, atau tanda hubung";
    if (!d.nama_pic) e.nama_pic = "Nama penanggung jawab wajib diisi";
    if (!d.phone) e.phone = "No. WhatsApp wajib diisi";
    else if (!/^[0-9+\-\s()]{8,20}$/.test(d.phone)) e.phone = "Nomor WhatsApp tidak valid";
    if (!d.email) e.email = "Email wajib diisi";
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(d.email)) e.email = "Email tidak valid";
    if (!d.setuju) e.setuju = "Centang pernyataan ini untuk melanjutkan";
    return e;
  }

  // Usulan alamat: ketik bebas, dirapikan jadi huruf kecil-angka-tanda hubung.
  const sub = $("#fSub");
  sub?.addEventListener("input", () => {
    const rapi = sub.value.toLowerCase().replace(/[\s_.]+/g, "-").replace(/[^a-z0-9-]/g, "");
    if (rapi !== sub.value) sub.value = rapi;
  });

  /* ===================== Kirim ===================== */

  function kumpulkan() {
    const v = (name) => String(form.elements[name]?.value || "").trim();
    const angka = (name) => {
      const s = v(name);
      return s === "" ? 0 : Math.trunc(Number(s));
    };
    return {
      paket: form.querySelector('input[name="paket"]:checked')?.value || "",
      periode: periodeTerpilih(),
      nama_usaha: v("nama_usaha"),
      jenis_usaha: v("jenis_usaha"),
      jumlah_gudang: angka("jumlah_gudang"),
      jumlah_pengguna: angka("jumlah_pengguna"),
      kota: v("kota"),
      alamat: v("alamat"),
      npwp: v("npwp").replace(/[^0-9]/g, ""),
      subdomain: v("subdomain").toLowerCase(),
      nama_pic: v("nama_pic"),
      jabatan: v("jabatan"),
      phone: v("phone"),
      email: v("email"),
      catatan: v("catatan"),
      setuju: !!form.elements.setuju?.checked,
      website: v("website"),
    };
  }

  form.addEventListener("submit", async (ev) => {
    ev.preventDefault();
    clearErrors();
    status.className = "form-status";

    const d = kumpulkan();
    const errs = validate(d);
    if (Object.keys(errs).length) {
      showErrors(errs);
      return;
    }

    const label = submit.innerHTML;
    submit.disabled = true;
    submit.innerHTML = '<i class="bi bi-hourglass-split"></i> Mengirim…';
    try {
      const res = await fetch("/api/daftar", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(d),
      });
      const json = await res.json().catch(() => ({}));
      if (res.ok && json.status) {
        tampilkanSelesai(json.data || {}, d);
        return;
      }
      if (json.errors) showErrors(json.errors);
      status.className = "form-status err";
      status.textContent = json.message || "Gagal mengirim pendaftaran. Silakan coba lagi.";
    } catch (e) {
      status.className = "form-status err";
      status.textContent = "Koneksi bermasalah. Silakan coba lagi.";
    } finally {
      submit.disabled = false;
      submit.innerHTML = label;
    }
  });

  /* ===================== Selesai ===================== */

  function tampilkanSelesai(r, d) {
    form.hidden = true;
    const done = $("#daftarDone");
    $("#doneUsaha").textContent = r.nama_usaha || d.nama_usaha;
    $("#doneKode").textContent = r.kode || "-";
    if (r.penawaran) {
      $("#doneStep2Judul").textContent = "Penawaran harga";
      $("#doneStep2").textContent = "Tim sales menyusun penawaran sesuai kebutuhan Anda dan mengirimkannya ke email Anda.";
    } else {
      const per = r.periode === "tahunan" ? "tahun pertama" : "bulan pertama";
      $("#doneStep2").textContent = `Tagihan ${per} paket ${r.paket} sebesar ${P.rupiah(r.estimasi)} (belum PPN) dikirim ke ${
        r.email || d.email
      }.`;
    }

    const cfg = window.HUB_CONTACT || {};
    const wa = String(cfg.whatsapp || "").replace(/[^0-9]/g, "");
    const btnWa = $("#doneWa");
    if (wa && btnWa) {
      const teks = `Halo, saya sudah mendaftar HUB dengan nomor ${r.kode} atas nama ${r.nama_usaha || d.nama_usaha}.`;
      btnWa.href = `https://wa.me/${wa}?text=${encodeURIComponent(teks)}`;
      btnWa.hidden = false;
    }

    done.hidden = false;
    $(".steps-mini")?.setAttribute("hidden", "");
    window.scrollTo({ top: 0 });
    done.focus({ preventScroll: true });
  }

  $("#doneCopy")?.addEventListener("click", async () => {
    const kode = $("#doneKode").textContent;
    const ic = $("#doneCopy i");
    try {
      await navigator.clipboard.writeText(kode);
      ic.className = "bi bi-check2";
      setTimeout(() => (ic.className = "bi bi-copy"), 1500);
    } catch (e) {
      // Clipboard tidak diizinkan (mis. bukan HTTPS): biarkan pengguna menyalin manual.
      const r = document.createRange();
      r.selectNodeContents($("#doneKode"));
      const s = window.getSelection();
      s.removeAllRanges();
      s.addRange(r);
    }
  });
})();
