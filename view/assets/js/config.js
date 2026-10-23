/* Kontak yang tampil di situs. ISI SEBELUM DITAYANGKAN.
 *
 * - whatsapp : nomor format internasional TANPA "+" / spasi, mis. "6281234567890".
 *              Kosong = tombol WhatsApp (kartu kontak & tombol melayang) disembunyikan.
 * - email    : alamat email sales/support. Kosong = tautan email disembunyikan.
 * - waText   : pesan pembuka yang otomatis terisi saat pengunjung membuka chat.
 *
 * Setelah mengubah berkas ini, naikkan ?v= pada tag <script> di index.html. */
window.HUB_CONTACT = {
  whatsapp: "",
  email: "",
  waText: "Halo, saya tertarik dengan HUB dan ingin menjadwalkan demo.",
};

/* Mode PRATINJAU = situs disajikan tanpa server Go (GitHub Pages), jadi
 * /api/demo & /api/daftar tidak ada. Form tetap bisa dicoba, tetapi TIDAK
 * dikirim ke mana pun; hasilnya disimulasikan dan ada spanduk di atas.
 * Otomatis aktif di *.github.io; di server Go (lokal / VPS) mati. */
window.HUB_PREVIEW = /\.github\.io$/.test(location.hostname);
