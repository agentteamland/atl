# Skill seçim disiplini

Aynı projede birden fazla ATL takımı yüklediğinde, asistanın her tur için seçebileceği birden fazla skill seti olur. ATL kasıtlı olarak takımları istem içeriğine göre **otomatik etkinleştirmez** — seçim asistanın yargısına bırakılır ve bu yargı senin verdiğin net istemlerle şekillenir. Bu sayfa o sözleşmenin kullanıcı tarafıdır.

## Kaputun altında ne oluyor?

Her ATL takımı kendi skill'lerini (asistanın çağırabileceği eğik-çizgi komutları) yayımlar. Aynı projeye örneğin bir yazılım takımı ve bir kişisel danışmanlık takımı kurduğunda, iki takımın skill'leri de bu projedeki her Claude Code oturumunda erişilebilir hale gelir. Asistan skill'lerin birleşimini görür ve tur başına bir tanesini seçer — ya da hiçbirini.

ATL'nin [`skill-selection-discipline` kuralı](https://github.com/agentteamland/core/blob/main/rules/skill-selection-discipline.md), asistanın skill listesinin tamamını gözden geçirmesini, istemin niyetini her skill'in amacıyla eşleştirmesini ve birden fazla skill aday olduğunda netleştirme yapmasını gerektirir. Kural her oturumda otomatik yüklenir.

## ATL neden takım skill'lerini otomatik tetiklemiyor?

Üç otomatik etkinleştirme mekanizmasını değerlendirdik:

- **Takım başına watch hook'ları** — yüklü her takım her istemi yoklar; maliyet takım sayısıyla doğrusal ölçeklenir.
- **Merkezi yönlendirici** — ATL tarafında tek bir gözlemci hangi takımın ilgili olduğuna dair ipucu verir; yine de her istemde çıkarımsal yönlendirme gerektirir.
- **Tembel takım yükleme + arabuluculuk** — küçük bir LLM çağrısı her istemi doğru takıma yönlendirir; deterministik ama her turda gecikme × maliyet ekler.

Üçü de aynı gerekçeyle reddedildi: hiçbir aday tam deterministik değil VE her aday önemsiz olmayan maliyet getiriyor. Skill seçimi asistanın işidir; istem netliğin asistanın işini güvenilir kılan şeydir.

## Senin yapabileceklerin

Asistanın doğru seçim yapmasına üç alışkanlık yardım eder:

### 1. Alanın hakkında spesifik ol

Muğlak bir istem asistanı tahmine zorlar; tahmin de yanlış-takım seçiminin başladığı yerdir.

| Muğlak | Spesifik |
|---|---|
| "Ne yapmam lazım?" | "Yarın Ahmet'le toplantım var, ne sormam lazım?" (açıkça kişisel) |
| "Şuna bir bak" | "API endpoint'inde validation yazalım" (açıkça yazılım) |
| "Yardım et" | "Bu hafta finansal durumumu gözden geçirelim" (açıkça kişisel danışmanlık) |

İstemin alanı ne kadar net adlandırırsa, asistanın çıkarım yapması o kadar azalır.

### 2. Skill'i biliyorsan adını söyle

Hangi eğik-çizgi komutunun çalışacağını biliyorsan, doğrudan onu çağırmak asistanın hatırlamasını beklemekten daha hızlı ve daha güvenilirdir. `/drain` yazmak anında çalışır; "öğrendiklerimi işle" niyetin skill adına eşlenmesini gerektirir.

### 3. Yanlış takım seçildiyse tur ortasında düzelt

Asistan yanlış takımın çerçevesinden cevap vermeye başladıysa (kişisel bir soruya yazılım yanıtı ya da tersi), söyle:

> "Bu konu personal değil, software" — asistan geçiş yapar ve düzeltmeyi oturum içinde hatırlar.

Tur ortası düzeltme normal ve hızlıdır. Yanlış skill'in baştan sona çalışması zaman maliyetinin gerçek kaynağıdır.

## Çoklu-takım projeleri üzerine bir not

ATL projelerinin çoğunda yalnızca tek bir takım gerekir. Çoklu-takım kurulumları çoğunlukla şu durumlarda ortaya çıkar:

- Asistanı hem kişisel danışmanlık işleri hem de belirli bir yazılım projesi için aynı `.atl/` altında kullanıyorsundur (ve iki ayrı Claude Code oturumu arasında bağlam değiştirmek istemiyorsundur).
- Bir temel takımı takıma özel eklentilerle genişletmişsindir (ör. `acme/example-team`'i genişleten `acme/proto-team`).

Asistanla bir istemin hangi takıma ait olduğunu sürekli tartışıyorsan, bu (a) ayrı projelere bölmenin (her biri tek takımla), ya da (b) istemlerinde daha açık olmanın sinyalidir. ATL çoklu-takım kurulumlarını desteklemek üzere tasarlanmıştır, ama görünmez olmak üzere tasarlanmamıştır — istemlerin yönlendirme mantığının bir parçasıdır.

## İlgili

- **Kural kaynağı:** [`core/rules/skill-selection-discipline.md`](https://github.com/agentteamland/core/blob/main/rules/skill-selection-discipline.md) — bu sayfanın kullanıcı-tarafı muadili olduğu asistan-tarafı kural.
- **Otomatik etkinleştirme gerekçesi:** [`auto-team-activation.md`](https://github.com/agentteamland/workspace/blob/main/.atl/brain-storms/auto-team-activation.md) — takım başına, merkezi yönlendirici ve tembel-yükleme arabuluculuk yaklaşımlarının neden reddedildiğini belgeleyen workspace brainstorm'u.
- **Karpathy ilkeleri:** [`/tr/guide/karpathy-guidelines`](/tr/guide/karpathy-guidelines) — reddi besleyen daha geniş davranış ilkeleri (Önce Sadelik).
