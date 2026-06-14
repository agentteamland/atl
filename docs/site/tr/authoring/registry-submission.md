# Kayıt defteri başvurusu

Takımını herkese açık kayıt defterinde listelemek ona kısa bir ad kazandırır (`atl install your-team` yerine `atl install https://github.com/…`). Kayıt defteri, PR ile bakım gören tek bir JSON dosyasıdır.

## Kayıt defteri nedir?

[`agentteamland/registry`](https://github.com/agentteamland/registry) tek bir dosya barındırır: `teams.json`. Her giriş, kısa bir adı bir Git URL'sine ve üst bilgisine eşler:

```json
{
  "teams": [
    {
      "name": "software-project-team",
      "url": "https://github.com/agentteamland/software-project-team",
      "status": "verified",
      "description": ".NET 9 API + Flutter + React + full Docker stack",
      "keywords": ["dotnet", "flutter", "react", "docker"]
    }
  ]
}
```

## Başvurmadan önce

Takımın şu özelliklere sahip olmalı:

1. **Git URL'sinden kurulabilir** — `atl install https://github.com/you/your-team.git` uçtan uca çalışıyor.
2. **Şema bakımından geçerli** — `team.json` doğrulamayı geçiyor. [Bir takım yazma](./creating-a-team) sayfasını izlediysen her depo bu denetimi CI'de zaten alır.
3. **Belgeli** — deponun `README.md` dosyası kullanıcıya takımın ne için olduğunu ve nasıl kullanılacağını anlatıyor.
4. **Etiketli** — en az bir SemVer etiketi (`v0.1.0` ya da üstü).

## Adımlar

1. `agentteamland/registry` deposunu **çatalla**.

2. Kaydını `teams.json` içindeki `teams` dizisine **ekle**. Diziyi `name` alanına göre alfabetik tut.

   ```json
   {
     "name": "your-team",
     "url": "https://github.com/you/your-team",
     "status": "community",
     "description": "One-sentence pitch. Shows up in atl search.",
     "keywords": ["what", "your", "team", "covers"]
   }
   ```

   Alanlar:
   - `name` — `team.json` içindeki `name` ile aynı olmalı.
   - `url` — Git HTTPS URL'si (sondaki `.git` zorunlu değil).
   - `status` — yeni başvurular `"community"` ile başlar. Bakımcılar inceleme sonrasında `"verified"` durumuna yükseltir.
   - `description` — kullanıcıya görünen tek satırlık tanıtım. **10-200 karakter** (şemada `description.maxLength = 200`). `team.json` içindeki `description` ile aynı değer. 200'ü aşmak, kayıt defteri PR'larının CI'de en sık başarısız olma sebebidir.
   - `keywords` — `atl search` eşleşmesi için.

3. **Push'tan önce yerelde doğrula.** Kayıt defteri deposu, CI'nin yaptığı çevrimdışı denetimleri yerelde çalıştıran bir betikle birlikte gelir:

   ```bash
   npm install -g ajv-cli ajv-formats   # tek seferlik; eğer ajv yüklü değilse
   ./scripts/validate.sh
   ```

   ::: tip git push'a bağla; geçersiz bir kayıt defterini asla push'layamayasın
   ```bash
   git config core.hooksPath .githooks   # klon başına bir kez
   ```
   Bundan sonra `teams.json` ya da `schemas/` dizinine dokunan her `git push`, `./scripts/validate.sh` betiğini kendiliğinden çalıştırır ve doğrulama başarısız olursa push'u iptal eder. 200 karakterlik `description` taşmasını başarısız bir PR denetimine bırakmak yerine yerelde yakalar.
   :::

4. **PR aç.** CI şunları doğrulayacak:
   - JSON şema uyumluluğu (`description` için 10-200 karakter aralığı dâhil).
   - `url`, `team.json` getirildiğinde 200 dönüyor mu.
   - Getirilen `team.json` takım şemasına göre geçerli mi.
   - `name` benzersizliği (kayıt defterinde yinelenme yok).

5. **İncelemeyi bekle.** Bakımcılar takımın var olduğunu, kurulduğunu ve söylediğini yaptığını denetler. Doğrulanmış durumu inceleme sonrasında verilir.

## Durum yaşam döngüsü

- **`community`** — listeli ve kurulabilir, henüz incelenmemiş. Her başvuru buradan başlar.
- **`verified`** — AgentTeamLand bakımcılarının incelediği durum; temiz kurulup sözleşmelere uyması beklenir. Bakımcılar bunu izleyen bir PR ile yükseltir.
- **`deprecated`** — artık bakım görmüyor. Hâlâ kurulabilir ama kullanıcılar bir uyarı görür. Tipik olarak yazar depoyu arşivlediğinde ya da takım bir yeniden yazımla değiştirildiğinde verilir.

## Bir takımı kaldırma

Takımının kayıt defterinden kaldırılmasını istiyorsan kaydını silen bir PR aç. Zaten kurmuş olan kullanıcılar Git URL'sini doğrudan kullanmayı sürdürebilir.

## Bir kaydı güncelleme

Kayıt defteri PR'larına yalnızca üst bilgi değişiklikleri girer — **sürüm değil**. Sürüm çözümü devingendir: `atl` kurulum / güncelleme anında takımın kendi etiketlerini okur. Yalnızca açıklaman, anahtar sözcüklerin, URL'n ya da durumun değiştiğinde bir kayıt defteri PR'ı gerekir.

## Sorular?

[`agentteamland/registry` üzerinde bir issue aç](https://github.com/agentteamland/registry/issues).

## İlgili

- **[Bir takım yazma](./creating-a-team)** — kayıt defterinden önceki kontrol listesi.
- **[team.json](./team-json)** — kayıt defterinin karşı doğruladığı şema.
