# Şema

ATL v2'de ayrı, makine tarafından okunabilir bir JSON Şema dosyası yoktur.

v1'de `team.json`, CI ortamında kontrol edilen bağımsız bir `team.schema.json` (JSON Schema Draft 2020-12) dosyasına karşı doğrulanıyordu. v2 bu dosyayı kaldırdı. `team.json` sözleşmesi artık yalnızca insanlar için belgelenmekte; CLI ise kurulum sırasında bunu en düşük düzeyde denetlemektedir.

## Sözleşmenin tek bir yeri var

**[`team.json`](/tr/authoring/team-json)** tam alan başvurusudur — her alan, türü, zorunlu olup olmadığı, ne anlama geldiği ve örneklerle birlikte burada yer alır.

## CLI'ın denetledikleri

`atl install` çalıştırdığında CLI herhangi bir JSON Şema doğrulayıcısı çalıştırmaz. Şu üç şeyi kontrol eder:

- `team.json` geçerli JSON olarak ayrıştırılabilmeli.
- `name` alanı bulunmalı.
- `agents[]`, `skills[]` ve `rules[]` altında bildirilen her varlık, diskte beklenen konumda mevcut olmalı.

Bu kontrollerden herhangi biri başarısız olursa kurulum hata vererek durur. Bunun dışındaki her şey (fazladan alanlar, biçimlendirme) yok sayılır.

## İlgili

- **[team.json](/tr/authoring/team-json)** — alan başvurusu ve örnekler.
- **[Sözlük](./glossary)** — ATL genelinde kullanılan terimler.
