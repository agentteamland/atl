# `atl store`

Kurulu takımların **bildirdiği kalıcı depolar** üzerindeki işlemler.

Kalıcı depo, bir takımın "burada hayatta kalması gereken içerik var" diye bildirdiği bir dizindir — `team.json` içindeki `capabilities.<ad>.store`, kurulum anında kaydedilir. Çekirdek hangi takımın hangi yolu sahiplendiğini asla öğrenmez; yalnızca bildirime uyar. Bugün birinci-parti tek örnek `~/.atl/profiles` altındaki profil deposudur.

## `atl store version`

Bildirilen her depoyu **yerel git** altına alır ve son anlık görüntüden bu yana değişen ne varsa işler.

```bash
atl store version
```

Bu, yazma politikası "son yazan kazanır" olan depolar için saklama tabanıdır: üzerine yazılan bir profil alanı, önceki değeri bir commit'te durduğu için geri alınabilir kalır.

**Bu, `atl session-start` komutunun otomatik olarak çalıştırdığı pasın aynısıdır** — aynı dizini farklı şartlarla versiyonlayan iki mekanizma yerine, tek uygulama ve iki tetik. Bu ayrım önemli, çünkü [`/profile-backup`](/tr/teams/profile-team) bunu çağırır: henüz depo olmayan bir dizin eskiden backup'ı reddettiriyordu ve kullanıcıya versiyonlamanın session-start'ın işi olduğu söyleniyordu — üzerine işlem yapamayacağı bir bilgi.

### Neyi yapmayı reddeder, ve neden

`atl store version` uygunluğa kendisi karar verir; bu yüzden elle çağrılması, kendiliğinden çalışan pas kadar güvenlidir:

- **Var olmayan depo oluşturulmaz.** Eksik bir dizin, o özelliğin bu makinede kullanılmadığı anlamına gelir; oluşturmak hem diski kirletir hem de özelliği etkinmiş gibi gösterir.
- **Boş depo başlatılmaz.** Geride yalnız bir `.git` bırakmak zararsız değildir — dizini boş olmaktan çıkarır, ve "çalışacak bir şey var mı" diye boşluğa bakan bir tüketici o zaman hiçbir şey tutmayan bir depo hakkında rapor verir.
- **Başka bir deponun içine yuvalanmış depoya dokunulmaz.** Orada başlatmak dıştaki depoyu gölgeler, commit atmak ise bu pasın sahibi olmadığı bir depoya yazmak olur.

### Çıktı

```
versioned 2 durable store(s)     # bir şey değişti ve işlendi
no-store-versioned               # uygun bir şey yok, ya da hiçbir şey değişmedi
```

Çağıranı asla başarısız etmez. Bir şey yapmadıkça sessiz kalan session-start pasının aksine, bu her durumda yazar — çünkü biri sordu.

## İlgili

- `atl session-start` — aynı versiyonlama pasını oturum başına bir kez otomatik çalıştırır.
- `/profile-backup` ve `/profile-restore` — makine dışı yarısı. Yerel git üzerine yazılan bir değeri geri alınabilir kılar; diski kaybetmeye karşı hiçbir şey yapmaz.
