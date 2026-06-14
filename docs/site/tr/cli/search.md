# `atl search`

Herkese açık kayıt defterinde arama yapar.

## Kullanım

```bash
atl search <query>
```

`<query>`, takım adlarına, açıklamalara ve anahtar sözcüklere karşı eşleşir. Büyük-küçük harf duyarsızdır ve alt-dize tabanlıdır; düzenli ifade değildir.

## Örnek

```bash
atl search dotnet
```

```
Found 1 team(s) matching "dotnet":

  software-project-team@1.2.1 [verified]
    .NET 9 API + Flutter + React + full Docker stack
    https://github.com/agentteamland/software-project-team
    keywords: dotnet, csharp, flutter, react
```

Durum rozetleri (`name@version` sonrasındaki köşeli parantez içinde):

- **`[verified]`** — AgentTeamLand bakımcıları tarafından incelenmiştir. Temiz kurulup sözleşmelere uyması beklenir.
- **`[community]`** — kayıt defterinde listelidir, henüz incelenmemiştir. Çalışır, ama kullanım riski sana aittir.
- **`[deprecated]`** — hâlâ kurulabilir ama artık bakım görmemektedir. Uygun olduğunda başkasına geç.

## Sorgu zorunludur

`atl search` tam olarak bir konumsal argüman ister. Sorgu vermeden çalıştırmak kullanım hatasıyla çıkar — kataloğun tamamına göz atmak için [GitHub'daki kayıt defterine](https://github.com/agentteamland/registry/blob/main/teams.json) bak ya da `atl search team` gibi geniş bir anahtar sözcük kullan.

## Çevrimdışı davranış

Kayıt defteri ilk çekimden sonra yerelde önbelleklenir. `atl search` önbelleklenmiş kopyayı kullanarak çevrimdışı çalışır; sonuçların bayat olabileceğini sana belirten bir not yazdırır.

## Sonuç yok mu?

Kayıt defteri PR güdümlü ve gençtir — alanın henüz kapsanmıyorsa büyük olasılıkla bu yalnızca "henüz değil" demektir. Seçenekler:

- Bir Git URL'sini doğrudan kullan: `atl install https://github.com/you/your-team.git`.
- Kendi takımını yayımla: [Bir takım yazma](/tr/authoring/creating-a-team).
- Kayıt defterine gönder: [Kayıt defteri başvurusu](/tr/authoring/registry-submission).

## İlgili

- [`atl install`](/tr/cli/install) — bulduğunu kur.
- [Kayıt defteri başvurusu](/tr/authoring/registry-submission) — takımını listele.
