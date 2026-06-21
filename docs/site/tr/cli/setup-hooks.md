# `atl setup-hooks`

ATL'nin otomasyon hook'larını Claude Code'a kur; böylece platform kendi kendini bakımlı tutar — senden hiçbir elle iş beklemez.

v2'de otomasyon **zorunludur, isteğe bağlı değil**: `atl install` bu hook'ları zaten senin için bağlar. `atl setup-hooks` komutunu doğrudan yalnızca hook'ları ayrıca (yeniden) kurmak istiyorsan ya da kısıtlama aralığını değiştirmek istiyorsan çalıştırırsın.

## Kullanım

```bash
atl setup-hooks                    # varsayılan 10 dakikalık tick kısıtlamasıyla kur
atl setup-hooks --throttle=5m      # oturum içinde daha sık tick (her 5 dakikalık etkinlikte)
atl setup-hooks --throttle=1h      # daha seyrek tick
```

`--throttle` yalnızca `UserPromptSubmit` hook'unu (oturum içi `atl tick`) etkiler. `SessionStart` her zaman tam olarak çalışır.

## Ne yapar?

`~/.claude/settings.json` dosyasına iki giriş yazar:

```json
{
  "hooks": {
    "SessionStart": [
      { "hooks": [
          { "type": "command", "command": "atl session-start" }
      ]}
    ],
    "UserPromptSubmit": [
      { "hooks": [
          { "type": "command", "command": "atl tick --throttle=10m" }
      ]}
    ]
  }
}
```

Claude Code şunları kendiliğinden çalıştırır:

### `SessionStart` — açılış zamanı bakımı

Yeni bir Claude Code oturumu açtığında bir kez çalışır. `atl session-start`, açılış zamanına ait işleri sırayla yapar:

1. **Platform çekirdeğini yansıt** — ikili içindeki kuralları ve becerileri global `~/.claude` katmanına yeniler; yüklü `atl` sürümüyle eş adım kalır.
2. **Önceki oturumu drenaj et** — bu projenin son drenajdan bu yana değişen transkriptlerini bulur, asistan metnini çıkarır ve satır içi `<!-- learning: ... -->` işaretçilerini `~/.atl/queue.db` adresindeki kalıcı kuyruğa tam olarak bir kez aktarır.
3. **Doctor öz-denetimi** — kuyruk sağlığı ile varlık bütünlüğü denetimlerini çalıştırır; sorunları yüzeye çıkarır ya da otomatik olarak iyileştirir.
4. **Bekleyen öğrenmeleri sinyal et** — kuyrukta işlenmemiş öğrenmeler varsa `atl: N learning(s) pending — run /drain to fold them into the knowledge base` tek satırını yazdırır; Claude bunu bilgi tabanına katlaması için bir ipucu olarak görür.

`SessionStart`, hook stdout çıktısını Claude'un bağlamına ileten tek Claude Code olayıdır; dolayısıyla `session-start` komutunun yazdığı her şey Claude'a ulaşır. Yüzeye çıkarılacak bir şey yoksa sessiz kalır; sıradan bir açılış hiçbir ek maliyete yol açmaz.

### `UserPromptSubmit` — kısıtlamalı oturum içi tick

Claude'a gönderdiğin her mesajdan önce çalışır. `atl tick --throttle=10m`, her komut çağrısında ucuz işleri, kısıtlama penceresi boyunca ise yalnızca bir kez daha ağır işleri yapar:

- **Fan-out** (her çağrıda, nesil korumalı) — global katman bu proje son fan-out'undan bu yana değiştiyse güncellenen varlıkları çeker. Aksi hâlde tek bir küçük dosya okumasıdır; her mesaja binecek kadar ucuzdur.
- **Drenaj + doctor** (kısıtlamalı) — bu projenin transkriptlerini yeni işaretçiler için yeniden tarar ve doctor öz-denetimini çalıştırır. Son tick kısıtlama penceresi içindeyse atlanır; mesaj başına maliyet tek bir dosya-bilgisi çağrısına düşer.
- **Kazanımları yükselt** (kısıtlamalı) — bu projenin birikmiş kazanımlarını global katmana taşır (katkılı, çakışma arşivleme, sabitlenebilir); elle `atl promote` çalıştırmayı beklemeden dolaşıma girer.

Bir şey yüzeye çıktığında Claude bağlamında ilgili satırı görür ve bunu kısaca belirtebilir. Hiçbir şey değişmediğinde hiçbir şey görmezsin.

## İşaretçi güdümlü öğrenme işlemesi Claude'a nasıl ulaşır?

Yakalama otomatiktir; yalnızca *bilgi tabanına katma* adımı bir Claude turu gerektirir (CLI'nin tek başına yapamayacağı LLM işi — CLI/Skill sınırı):

```
[N. oturumu kapatırsın]   satır içi öğrenme işaretçileri transkript dosyasında oturur
        ↓
[N+1. oturumu açarsın]
        ↓
SessionStart hook tetiklenir → atl session-start
        → önceki oturumun transkriptlerini ~/.atl/queue.db'ye drenaj eder (her işaretçi tam olarak bir kez kuyruğa girer)
        → kuyrukta bekleyen öğrenmeler varsa `atl: N learning(s) pending — run /drain ...` yazdırır
        ↓
Claude Code, stdout çıktısını Claude'un ilk additionalContext alanına enjekte eder
        ↓
[N+1. oturumda ilk turun]
        ↓
Claude sayıyı görür, /drain komutunu çağırır
        ↓
/drain, kuyruktaki her öğrenmeyi wiki / journal / ajan KB'ye katar, ardından siler (ack)
```

Tek bir oturum içinde `atl tick`, mesajlar arasında kuyruğu güncel tutar; böylece bir sonraki `session-start`'ta (ya da bir sonraki `atl learnings` çağrısında) görülen sayı her zaman güncel olur.

İşaretçi biçimi ve kuyruk durum/peek/ack yüzeyi için [`atl learnings`](/tr/cli/learnings) sayfasına, oturum içi döngü için [`atl tick`](/tr/cli/tick) sayfasına ve kuyruktaki öğrenmelerin bilgi tabanına nasıl katıldığına ilişkin ayrıntılar için [`/drain` becerisi](/tr/skills/drain) sayfasına bak.

## Neden bu iki hook?

| Hook | Yanıtladığı soru |
|---|---|
| `SessionStart` (`atl session-start` üzerinden) | "Claude Code'u taze açıyorum — son oturumun geride bıraktıklarını drenaj et, bozuk bir şey varsa iyileştir, katlanacak öğrenmeler varsa haber ver." |
| `UserPromptSubmit` (`atl tick` üzerinden) | "Bu oturumda bir süredir çalışıyorum — kuyruğu güncel tut, global katman değişikliklerini çek ve kazanımları mesajlar arasında, ucuzca dolaşıma sok." |

İkisi birlikte üç hızlı oturum içi döngüyü hayata geçirir: her mesajda fan-out, kısıtlamalı tick ve açılış zamanı drenajı.

## İdempotenlik — yeniden çalıştırması güvenlidir

Birleştirme, sahip olduğun diğer hook'ları korur. `atl setup-hooks` yeniden çalıştırıldığında (ya da aynı hook'ları bağlayan `atl install` çalıştırıldığında) yalnızca `atl`'ye ait girişlere dokunur — `atl ` ön ekiyle başlayan her komut. `settings.json` dosyasındaki diğer tüm hook'lar, izinler, model ayarları ve `extraKnownMarketplaces` el değmemiş kalır. Yazma işlemi atomiktir.

## Bunu ne zaman çalıştırmalısın?

- Etkileşimli Claude Code kullanıcıları için **her zaman** — `atl install` zaten yapar ama kısıtlamayı değiştirmek için yeniden çalıştırabilirsin.
- CI / betikli kullanım için **önerilmez** (hook'lar CI'da gereksiz yere tetiklenir).

## Çevrimdışı davranış

Hook'lar yalnızca yerel dosyaları okur ve yazar — transkriptleri drenaj etmek, bbolt kuyruğu ve doctor denetimleri ağ bağlantısı gerektirmez. Bir hook çalışmanı asla engellememelidir; bu yüzden `session-start` ve `tick` oturumu başarısız kılmaz; bir şeyler ters giderse bir satır çıkarır (ya da sessiz kalır) ve komut istemi normal biçimde sürer.

## İlgili

- [`atl tick`](/tr/cli/tick) — oturum içi bakım tick'i (`UserPromptSubmit` hook'unun çağırdığı komut)
- [`atl learnings`](/tr/cli/learnings) — kalıcı öğrenme kuyruğunu incele (status / peek / ack)
- [`atl doctor`](/tr/cli/doctor) — hook'ların her geçişte çalıştırdığı öz-denetim
- [`atl install`](/tr/cli/install) — ilk kurulum (bu hook'ları senin için bağlar)
- [CLI'yi kur](/tr/guide/install) — `atl`'yi makinene almak
