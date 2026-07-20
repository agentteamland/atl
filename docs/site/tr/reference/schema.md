# Şema

ATL v2'de ayrı, makine tarafından okunabilir bir JSON Şema dosyası yoktur.

v1'de `team.json`, CI ortamında kontrol edilen bağımsız bir `team.schema.json` (JSON Schema Draft 2020-12) dosyasına karşı doğrulanıyordu. v2 bu dosyayı kaldırdı. `team.json` sözleşmesi artık yalnızca insanlar için belgelenmekte; CLI ise kurulum sırasında bunu en düşük düzeyde denetlemektedir.

## Sözleşmenin tek bir yeri var

**[`team.json`](/tr/authoring/team-json)** tam alan başvurusudur — her alan, türü, zorunlu olup olmadığı, ne anlama geldiği ve örneklerle birlikte burada yer alır.

## CLI'ın denetledikleri

`atl install` çalıştırdığında CLI herhangi bir JSON Şema doğrulayıcısı çalıştırmaz. Şu üç şeyi kontrol eder:

- `team.json` geçerli JSON olarak ayrıştırılabilmeli.
- `name` alanı bulunmalı.
- Bir varlık dizini (`agents/`, `skills/`, `rules/`, `knowledge/`, `backends/`, `scripts/`, `packs/`) altında en az bir dosya göndermeli.

Bu kontrollerden herhangi biri başarısız olursa kurulum hata vererek durur. Bildirilen tek tek `agents[]`/`skills[]`/`rules[]` girişleri katalog üst verisidir ve kurulum sırasında diske karşı doğrulanmaz — bildirilen `agents[]` ve `skills[]` girişlerini, birinci taraf takımlar için `atl skills check` geliştirici komutu çapraz kontrol eder. Bunun dışındaki her şey (fazladan alanlar, biçimlendirme) yok sayılır.

## İlgili

- **[team.json](/tr/authoring/team-json)** — alan başvurusu ve örnekler.
- **[Sözlük](./glossary)** — ATL genelinde kullanılan terimler.
