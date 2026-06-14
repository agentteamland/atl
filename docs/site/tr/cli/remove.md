# `atl remove`

Mevcut projeden bir takımı kaldır.

## Kullanım

```bash
atl remove <team>             # etkileşimli — yerel değişiklikleri yok etmeden önce sorar (atl ≥ 1.0.0)
atl remove <team> --force     # etkileşimsiz — onay sorusunu atlar
```

`<team>` takımın kayıt defteri adıdır (URL değil). Kurulu adları `atl list` ile görebilirsin.

## Örnek

```bash
atl remove starter-extended
```

## Ne olur?

1. Takım `.claude/.team-installs.json` içinde bulunur.
2. Bu takımın `.claude/agents/`, `.claude/skills/`, `.claude/rules/` dizinlerine kurduğu her proje-yerel kopya, manifesto dosyasının kaynak başına SHA-256 referans hat'larına bakılarak belirlenir.
3. **Değişiklik denetimi**: her kaynağın güncel SHA-256'sı kurulum zamanındaki referans hat ile karşılaştırılır. Herhangi bir kaynak yerel olarak değiştirilmişse CLI `⚠ N kaynakta yerel değişiklik var` özetini yazdırır ve onay ister. `--force` bu soruyu atlar.
4. **Manifesto güdümlü izinli liste**: yalnızca bu takımın kaydettiği dosyalar silinir. `.claude/` altındaki kullanıcı tarafından yazılmış dosyalar (kendiliğinden büyüyen `children/` ve `learnings/`, özel beceriler, journal kayıtları, wiki sayfaları dahil) **korunur** — `atl` ile kaydedilmemiş oldukları için kaldırma işleminden sağ çıkarlar.
5. `.claude/.team-installs.json` atomik biçimde (geçici dosya + yeniden adlandırma) güncellenir.
6. Paylaşılan önbelleğe **dokunulmaz**. Takımın Git klonu yeniden kullanım için `~/.claude/repos/agentteamland/` altında kalır. Disk alanı geri kazanmak için önbellek dizinini elle sil.

::: warning Kaldırma sırasında kalıtım zorunlu kılınmaz
`atl remove`, kurulu başka bir takımın `extends` ile başvurduğu bir takımı kaldırmayı reddetmez. Bir alt takım hâlâ üst takıma başvuruyorken üst takımı kaldırırsan, alt takımın etkin kaynak kümesi bir sonraki `atl update` ya da `atl list`'te tutarsız hâle gelir. Önce `atl list` ile kalıtım zincirine bak — alt takımları üst takımlardan önce kaldır.
:::

## Bayraklar

| Bayrak | Etkisi |
|---|---|
| `--force` | Yerel değişiklikleri olan projeler için değişiklik-denetimi onay sorusunu atlar. CI ya da betikli sökme işleri için kullanışlıdır. |

## Örnek — CI'da zorlu kaldırma

```bash
atl remove software-project-team --force
```

`--force`'u etkileşimsiz bağlamlarda (CI, betikle sökme) kullan. Etkileşimli kullanımda varsayılanı tercih et — soru seni, `/save-learnings` ile saatlerce büyütülmüş içeriği ya da elle yapılmış düzenlemeleri kazara atmaktan korur.

## v1.0.0 öncesinden bu yana davranış değişiklikleri

`atl v1.0.0` öncesinde `atl remove` bir sezgisel yöntem kullanırdı ve takımın kaynaklarıyla birlikte kullanıcı tarafından yazılmış dosyaları kazara silebiliyordu. v1.0.0'daki manifesto güdümlü izinli liste, bu örtük hatayı kapattı — silinen her dosya, takımın açıkça kurduğu bir dosyadır.

Etkileşimli onay sorusu ve `--force` bayrağı da v1.0.0 ile geldi (öncesinde koşulsuz biçimde yıkıcıydı).

## İlgili

- [`atl list`](/tr/cli/list) — neyi kaldırabileceğini gör.
- [`atl install`](/tr/cli/install) — fikrini değiştirirsen yeniden kur.
- [`atl update`](/tr/cli/update) — değiştirilmemiş kopyaları kendiliğinden yeniler; `--force` ile kaldırıp kaldırmamaya karar verirken bilmen gereken bir şey (bir süredir dokunmadığın bir kopyanın değiştirilmemiş olduğunu fark etmeyebilirsin).
