# Sıkça sorulan sorular

### `atl` neyin kısaltmasıdır?

**A**gent**T**eam**L**and. CLI, organizasyon ve ekosistem aynı adı paylaşır.

### `atl`, Claude Code'un yerine geçer mi?

Hayır. `atl`, Claude Code'un `.claude/` dizininden zaten okuduğu dosyalar için bir teslim katmanıdır. Claude Code'u her zamanki gibi çalıştırmaya devam edersin; `atl` yalnızca sağlam bir yapılandırmayı yerine koymayı kolaylaştırır.

### Bu, dosyaları projeler arasında elle kopyalamaktan nasıl farklı?

Üç biçimde:

1. **Sürümleme.** Takımlar SemVer ile sürüm yayımlar. Bir sürüme sabitlersin; güncellemeler `atl update` üzerinden seçimlik gelir.
2. **Kalıtım.** Başkasının takımını çatallamadan onun üstüne inşa edebilirsin. Yazar iyileştirmeler yayımladığında sen onları miras alırsın.
3. **Erişim.** Bir takımı bir kez yayımlamak, `atl install your-team` çalıştıran her Claude Code kullanıcısına ulaştırır.

### Claude Code'u kullanmak için `atl` çalıştırmam gerekir mi?

Hayır. Claude Code `atl` olmadan da iyi çalışır. `atl`'yi, yeniden üretilebilir ve paylaşılabilir bir kurulum istediğinde kullan — elle hazırlanmış bir `.claude/` ile tek kişilik projeler de tümüyle geçerlidir.

### Aynı projede birden çok takım kurabilir miyim?

Evet. Her kurulum kendi kopyalarını `.claude/` altına ekler. İki takım aynı adda bir ajan yayımlıyorsa **ikinci kurulanın** sürümü kazanır (ilk kopyanın üzerine yazılır). Her kopyanın hangi takıma ait olduğunu `atl list` ile görebilirsin.

### Özel Git depolarından takım kurabilir miyim?

Evet, Git istemcin depoyu klonlayabildiği sürece (SSH anahtarları, PAT vb.). `atl` Git kimlik bilgilerini kendi yönetmez — kabuğunun Git kurulumuna devreder.

### `atl` sürümüm bir takım için çok eskiyse ne olur?

Takımın `team.json` dosyası bir `requires.atl` alt sınırı bildirebilir. Kurulu `atl` daha eskiyse, sana yükseltmeni söyleyen açık bir hata alırsın. Hangi kanaldan kurduysan ona göre `brew upgrade atl` (ya da eşdeğeri) çalıştır.

### Bir takımı nasıl eskiye düşürürüm?

```bash
atl install team-name@1.2.0   # belirli bir sürümü kur
```

Bu komut, kurulu olanı sabitlenen sürümle değiştirir.

### Bir projeyi silersem önbelleğe ne olur?

Hiçbir şey — önbellek (`~/.claude/repos/agentteamland/`) projeler arasında paylaşılır. Diğer projeler çalışmaya devam eder. Bir takımın artık hiçbir projede kullanılmadığından eminsen, ilgili önbellek dizinini elle silerek disk alanı geri kazanabilirsin.

### Bir takımı `atl` olmadan, elle kurabilir miyim?

Evet. Takım deposunu `~/.claude/repos/agentteamland/<team-name>/` altına klonla ve `.claude/` içindeki kopyaları kendin oluştur. CLI; bunun yanında kalıtım ve dışarıda bırakma çözümünü de otomatikleştirmek için vardır; çıktısında sihirli bir şey yoktur.

### `atl` kurulu takımlar listesini nerede tutar?

Her projede: `.claude/.team-installs.json`. İnsan tarafından okunabilir bir JSON dosyasıdır. Ne yaptığını biliyorsan düzenleyebilirsin — ama `atl install` / `atl remove` her zaman daha güvenlidir.

### `atl` telemetri gönderir mi?

Hayır. `atl` yerel bir araçtır: GitHub'dan klon yapar, kayıt defterini çeker, kopyalar yazar. "Eve telefon" yoktur.

### Bu bir Anthropic ürünü mü?

Hayır. AgentTeamLand, Anthropic'in Claude Code'uyla çalışan bağımsız bir açık kaynak projedir. MIT lisanslıdır. Ticari bağlantı yoktur.

### Nasıl katkıda bulunabilirim?

- Kayıt defterine **bir takım yayımla** — [Kayıt defteri başvurusu](/tr/cli/publish).
- **CLI'yı iyileştir** — PR'lar [`agentteamland/cli`](https://github.com/agentteamland/cli) deposunda beklenir.
- **Bu belgeleri iyileştir** — PR'lar [`agentteamland/docs`](https://github.com/agentteamland/docs) deposunda beklenir. Her sayfanın "Edit this page on GitHub" bağı vardır.
- **Issue aç** — hata raporları ve özellik istekleri ilgili depoya.

### Sorum burada yok.

[`agentteamland/docs` issues](https://github.com/agentteamland/docs/issues) üzerinde `faq` etiketiyle bir issue aç. Yaygın bir soruysa bu sayfaya eklenir.
