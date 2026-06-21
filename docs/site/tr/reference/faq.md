# Sıkça sorulan sorular

### `atl` neyin kısaltmasıdır?

**A**gent**T**eam**L**and. CLI, organizasyon ve ekosistem aynı adı paylaşır.

### `atl`, Claude Code'un yerine geçer mi?

Hayır. `atl`, Claude Code'un `.claude/` dizininden zaten okuduğu dosyalar için bir teslim katmanıdır. Claude Code'u her zamanki gibi çalıştırmaya devam edersin; `atl` yalnızca sağlam bir yapılandırmayı yerine koymayı kolaylaştırır — ve bunu kendi kendine iyileştirmeye devam eder.

### Bu, dosyaları projeler arasında elle kopyalamaktan nasıl farklı?

Üç biçimde:

1. **Sürümleme.** Takımlar SemVer sürümleri etiketler. `atl update`, kurulu her takımın en son yayımlanmış sürümünü çeker; böylece iyileştirmeler tüm kurulumlarına yayılır.
2. **Kapsam.** Bir takım bir [kapsama](/tr/guide/concepts#scope-global-and-project) kurulur — global (`~/.claude/`) veya proje (`<proje>/.claude/`). Her iki katmanda aynı adda bir varlık varsa, proje kopyası globali gölgeler. Elle kopyalanan dosyalarda böyle bir eksen yoktur.
3. **Kendi kendine çalışan bir öğrenme döngüsü.** Ajanların çalışırken kazanımlar biriktirirler — yeni öğrenmeler, keskinleşen beceriler. `atl`, bunları otomatik olarak dayanıklı bir kuyruğa aktarır; `/drain` her birini ajan bilgi tabanlarına katar ve `atl promote` / `atl publish` bunları dışa yayar. Elle kopyalanan dosyalar kendi kendine iyileşmez.

### Claude Code'u kullanmak için `atl` çalıştırmam gerekir mi?

Hayır. Claude Code `atl` olmadan da iyi çalışır. `atl`'yi, yeniden üretilebilir ve paylaşılabilir bir kurulum istediğinde kullan — elle hazırlanmış bir `.claude/` ile tek kişilik projeler de tümüyle geçerlidir.

### Aynı projede birden çok takım kurabilir miyim?

Evet. Her kurulum kendi kopyalarını `.claude/` altına ekler. İki takım aynı adda bir ajan yayımlıyorsa **en son kurulanın** sürümü kazanır (önceki kopyanın üzerine yazılır) ve `atl` tek satırlık bir uyarı yazdırır. Bu, kalıtım değil çakışma yönetimidir — her takım bağımsız olarak kurulur. Hangi kapsamda ne kurulu olduğunu görmek için [`atl list`](/tr/cli/list) kullan.

### Takımlar nereden geliyor? Özel bir depodan veya Git URL'sinden kurulum yapabilir miyim?

`atl install` **yalnızca katalog üzerinden** çalışır. Bir `<kullanıcı>/<takım>` referansı alır, bunu GitHub destekli kataloga karşı çözer ([`atl-team`](https://github.com/topics/atl-team) konusuyla etiketlenen genel depolardan oluşturulur), kaynağı kısa ömürlü bir HTTPS tarball olarak çeker ve takımın `agents/`, `skills/` ile `rules/` dizinlerini kapsamın `.claude/` içine kopyalar.

Özel depodan, rastgele bir Git URL'sinden, SSH'tan veya yerel dosya yolundan kurulum yapılamaz — bunlar v1'e aitti. Bir takımı kurulabilir hale getirmek için deposunu genel yap ve `atl-team` etiketiyle etiketle (ya da depodan [`atl publish`](/tr/cli/publish) çalıştır). Kataloğun nasıl sorgulandığını görmek için [`atl search`](/tr/cli/search) sayfasına bakabilirsin.

### `atl` sürümüm bir takım için çok eskiyse ne olur?

Takımın `team.json` dosyası, beklediği sürümü bildirmek için bir `requires.atl` alt sınırı tanımlayabilir. Güncel kalmak için `atl update` çalıştır (aynı zamanda `atl` ikili dosyasının paketlenmiş çekirdeğini de günceller) ya da kurulum betiğini yeniden çalıştır:

```bash
curl -fsSL https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.sh | sh
```

### Bir projeyi silersem `atl`'nin disk üzerindeki durumuna ne olur?

Global hiçbir şey etkilenmez. `atl` iki tür durum tutar ve bunlar ayrıdır:

- **Takım varlıkları** Claude Code'un kendi `.claude/` dizinlerinde yaşar. Bir projeyi silmek o projenin `.claude/` dizinini siler; `~/.claude/` içindeki global varlıklar etkilenmez.
- **`atl`'nin kayıt bilgileri** `~/.atl/` (global) ve `<proje>/.atl/` (proje) altında yaşar — önbelleğe alınmış katalog (`index.json`), öğrenme kuyruğu (`queue.db`), sabitlemeler ve takım başına kurulum bildirimleri. Global `~/.atl/` kalır; projenin `<proje>/.atl/` dizini projeyle birlikte silinir.

Temizlenecek paylaşımlı bir klon önbelleği yoktur — kaynaklar kurulum sırasında tek kullanımlık tarball olarak çekilir, disk üzerinde tutulmaz.

### Bir takımı `atl` olmadan, elle kurabilir miyim?

Evet — `atl` yalnızca bir kopyalamayı otomatikleştirir. Takım deposunu çek, ardından `agents/`, `skills/` ve `rules/` dizinlerini hedef `.claude/` içine kendin kopyala. Çözümlenecek kalıtım veya dışarıda bırakma mantığı yoktur ve doldurulacak kalıcı bir önbellek de bulunmaz. Kaybedeceğin tek şey `atl`'nin yazdığı kurulum bildirimidir (aşağıya bak); `atl update` ve `atl doctor` yenileme ve öz-onarım için bu bildirimlere güvenir.

### `atl` kurulu takımlar listesini nerede tutar?

Takım başına bir bildirim dosyasında, kapsam başına bir JSON dosyası olarak: `<kapsam>/.atl/installed/<kullanıcı>__<ad>.json` (`<kapsam>`, global kurulumda `~/.atl`, proje kurulumunda `<proje>/.atl`'dir). Her bildirim takımın `handle`, `name`, `version`, `scope`, `source` değerlerini, kurulum zamanını ve yazılan her dosya yolunun SHA-256 özetini içeren bir `files` haritasını kaydeder. `atl update`'in otomatik yenileme ve `atl doctor`'ın bütünlük denetimi bu özetleri okur. Bildirimleri elle düzenlemek desteklenmez — `atl install` / `atl remove` kullan.

### `atl` telemetri gönderir mi?

Hayır. `atl` yerel bir araçtır: takımları HTTPS üzerinden çeker, kataloğu okur ve kopyaları `.claude/` içine yazar. "Eve telefon" yoktur.

### Bu bir Anthropic ürünü mü?

Hayır. AgentTeamLand, Anthropic'in Claude Code'uyla çalışan bağımsız bir açık kaynak projedir. MIT lisanslıdır. Ticari bağlantı yoktur.

### Nasıl katkıda bulunabilirim?

- **Bir takım yayımla.** Takımın deposunu [`atl-team`](https://github.com/topics/atl-team) konusuyla etiketle ya da depodan [`atl publish`](/tr/cli/publish) çalıştır. Katalog bunu otomatik olarak alır — kayıt defteri yoktur ve başvuru PR'ı gerekmez.
- **CLI'yı, çekirdek kuralları/becerileri, birinci taraf takımları veya bu belgeleri iyileştir.** Hepsi [`agentteamland/atl`](https://github.com/agentteamland/atl) monoreposunda yaşar. PR'lar beklenir; her belge sayfasının "Edit this page on GitHub" bağı vardır.
- **Issue aç.** Hata raporları ve özellik istekleri [`agentteamland/atl`](https://github.com/agentteamland/atl/issues) deposuna gider.

### Sorum burada yok.

[`agentteamland/atl`](https://github.com/agentteamland/atl/issues) üzerinde `faq` etiketiyle bir issue aç. Yaygın bir soruysa bu sayfaya eklenir.
