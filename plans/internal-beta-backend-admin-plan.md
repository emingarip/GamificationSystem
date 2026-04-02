# Backend + Admin Internal Beta Plan - Status Revision

Updated after code audit on `2026-03-23`.

## Summary

Bu dokuman artik sadece hedef plan degil, mevcut implementasyonun plana gore gercek durumunu da kaydeder.

Net sonuc:

- Orijinal plan `kismen` uygulanmis.
- Reward layer, analytics endpoint'leri ve test event endpoint'i eklenmis.
- Proje halen orijinal `internal beta` bitis cizgisine ulasmis degil.
- En kritik eksikler veri sahipligi tutarsizligi, admin build kirikligi ve auth sertliginin yapilmamis olmasi.

Bu dosyanin amaci bundan sonra `neyin tamam oldugunu`, `neyin eksik veya yanlis kaldigini` ve `bitirmek icin ne yapilacagini` tek yerde net tutmaktir.

## Tamamlanan Kisimlar

- `RewardLayer` eklendi; `award_points` ve `grant_badge` icin Neo4j reward action yazimi ve Redis tabanli action idempotency altyapisi baslatildi.
- `POST /api/v1/events/test` endpoint'i eklendi.
- `GET /api/v1/analytics/summary`, `GET /api/v1/analytics/activity` ve `GET /api/v1/analytics/points-history` endpoint'leri eklendi.
- Kullanici profil response'una `rich_badge_info` ve `recent_activity` alanlari eklendi.
- Health check Redis ve Neo4j baglantisini gercekten kontrol eder hale getirildi.
- Rules ekraninda test event modal akisi eklendi.

Bu maddeler gercekten repo icinde mevcut:

- `internal/muscle/engine/reward.go`
- `internal/muscle/api/server.go`
- `internal/muscle/neo4j/client.go`
- `admin/src/pages/Rules.tsx`

## Eksik veya Yanlis Kalan Kisimlar

### 1. Reward ve source-of-truth modeli tamam degil

Orijinal planin hedefi `Neo4j = canonical source-of-truth`, `Redis = cache/operational state` idi.

Su anki durum bu hedefe uymuyor:

- Kullanici profili badge ve points state'ini hala Redis'ten okuyor.
- Manuel badge atama hala sadece Redis'e yaziyor.
- Badge CRUD hala Redis uzerinden yapiyor.
- Reward layer otomatik badge ownership'i Neo4j'ye yaziyor.
- Sonuc olarak manuel akislari ve otomatik akislari farkli veri tabanlari yonetiyor.

Bu durum `split-brain` riski olusturuyor.

Acik sorunlar:

- `GET /api/v1/users/{id}` Neo4j yerine Redis badge/points state'ine bagli.
- `POST /api/v1/users/{id}/badges` Neo4j ownership olusturmuyor.
- `POST/PUT/DELETE /api/v1/badges` Neo4j kataloguna tasinmamis.
- Badge duplicate name/id validation yok.

Etkilenen dosyalar:

- `internal/muscle/api/server.go`
- `internal/muscle/api/admin_compat.go`
- `internal/muscle/redis/client.go`
- `internal/muscle/engine/reward.go`

### 2. Reward execution dogrulugu hala zayif

Reward layer eklenmis olsa da motor akisinda kritik bir hata var:

- `engine.executeAction` reward layer error'unu kontrol etmiyor.
- Hata olsa bile motor legacy `RecordUserAction` yazmaya devam ediyor.
- Bu durumda reward persistence basarisiz olsa bile action history basarili gibi gorunebilir.

Ek olarak:

- `ProcessEventIdempotently` ve `MarkEventProcessed` fonksiyonlari yazilmis ama ana event isleme akisinda kullanilmiyor.
- Su an sadece action bazli idempotency kismen uygulanmis durumda.
- Bu da event-level dedupe hedefinin tamamlanmadigi anlamina gelir.

Etkilenen dosyalar:

- `internal/muscle/engine/engine.go`
- `internal/muscle/engine/reward.go`
- `internal/muscle/neo4j/client.go`

### 3. API sozlesmesi temiz degil

Orijinal plan:

- `GET /api/v1/users/{id}` icindeki `badges` alani dogrudan zengin `UserBadgeInfo` tipine donusmeliydi.

Su anki durum:

- `badges` alani hala ham `[]models.UserBadge`.
- Zengin veri ayri bir `rich_badge_info` alani olarak eklenmis.
- Frontend bunu cast ve fallback mantigi ile kullaniyor.

Bu calisir, ama hedeflenen temiz API degildir.

Ek API uyumsuzlugu:

- Frontend `testEvent` request'i `{ type, user_id, data }` seklinde gonderiyor.
- Backend `MatchEvent` bekliyor: `event_id`, `event_type`, `match_id`, `team_id`, `player_id`, `minute`, `timestamp`, `metadata`.
- Bu nedenle test event UI akisi sozlesme olarak yanlis.

Etkilenen dosyalar:

- `internal/muscle/api/models.go`
- `internal/muscle/models/event.go`
- `admin/src/lib/api.ts`
- `admin/src/pages/Rules.tsx`

### 4. Analytics gercek ama tamam degil

Ilerleme var, ama plan tamamen bitmis sayilamaz:

- Analytics endpoint'leri gercek backend verisi donduruyor.
- Fakat frontend hala `activeUsers` ve `totalRules` gibi degerleri uyduruyor.
- `GetTotalBadges` katalog badge sayisi yerine kullanicilar tarafindan kazanilmis badge sayisini hesapliyor.
- `points-history` endpoint'i `period` parametresini gercek anlamda uygulamiyor; query sabit toplamlama yapiyor.

Bu nedenle analytics `tam gercek veri` modunda degil, `kismen turetilmis` durumda.

Etkilenen dosyalar:

- `internal/muscle/api/server.go`
- `internal/muscle/neo4j/client.go`
- `admin/src/lib/api.ts`
- `admin/src/pages/Analytics.tsx`
- `admin/src/pages/Dashboard.tsx`

### 5. Admin panel internal beta seviyesinde degil

En net kanit:

- `npm run build` su an gecmiyor.

Gorulen problemler:

- kullanilmayan degiskenler ve stale placeholder import'lari var
- Dashboard yanlis parametre ile `getPointsHistory` cagiriyor
- Analytics ve Dashboard icinde eski mock/placeholder kalintilari var

Bu, admin panelin hala `releaseable internal beta` seviyesinde olmadigini gosterir.

Etkilenen dosyalar:

- `admin/src/lib/api.ts`
- `admin/src/pages/Analytics.tsx`
- `admin/src/pages/Dashboard.tsx`

### 6. Minimum internal auth hedefi yapilmamis

Orijinal plan:

- seeded veya env-driven tek admin kullanici
- hashed password dogrulamasi
- mevcut JWT middleware'in korunmasi

Su anki durum:

- herhangi bir e-posta ve 6+ karakter sifre ile admin token uretiliyor
- gercek kullanici deposu yok
- hashed password kontrolu yok

Bu nedenle auth hala `dev convenience` seviyesinde.

Etkilenen dosya:

- `internal/muscle/api/admin_compat.go`

### 7. Operational hardening tamam degil

Ilerleme:

- Redis ve Neo4j health check var

Eksik veya yanlis kisimlar:

- Kafka health check gercek probe degil; simdilik dogrudan `healthy` donuyor
- runbook/seed-reset dogrulama akisi kodla senkron olacak sekilde netlestirilmemis
- reward/action log standardizasyonu tam olarak kapanmis degil

Etkilenen dosyalar:

- `internal/muscle/api/server.go`
- `DOCKER_README.md`

## Revize Edilmis Kalan Is Listesi

Bu projeyi gercekten planlanan `internal beta` seviyesine cikarmak icin kalan isler asagidaki sirayla kapatilacak.

### 1. Veri sahipligini tekille

- Badge katalogunu Neo4j'ye tasi.
- Badge CRUD'u Neo4j repository uzerinden calistir.
- Manuel badge assign akisini Neo4j ownership ile hizala.
- Kullanici profile points ve badge state'ini canonical store'dan oku.
- Redis'i sadece cache ve operational state icin kullan.

Bitis kriteri:

- Ayni badge/points bilgisi manuel akista da otomatik akista da tek kaynaktan gelir.

### 2. Reward execution dogrulugunu tamamla

- `ExecuteRewardAction` error'unu engine katmaninda handle et.
- Reward basarisizsa legacy action kaydi yazma.
- `RecordUserAction` ile `RecordRewardAction` cakisiyorsa tek modele indir.
- Event-level dedupe'yi ana process akisina bagla.

Bitis kriteri:

- Reward basarisiz oldugunda sistem basarili gorunmez.
- Ayni event ikinci kez islendiyse ikinci kez yazim yapilmaz.

### 3. API ve admin sozlesmesini temizle

- `badges` alanini dogrudan zengin response'a donustur veya frontend'in kullanacagi resmi field'i tekille.
- `testEvent` frontend request modelini backend `MatchEvent` modeli ile hizala.
- `rich_badge_info` gibi gecici bridge alanlarini ya resmi contract yap ya da kaldir.

Bitis kriteri:

- Frontend cast/fallback mantigi olmadan response'u tuketir.

### 4. Admin build ve placeholder kalintilarini temizle

- `npm run build` gecmeli.
- Dashboard ve Analytics ekranlarindaki eski mock/placeholder kodlar temizlenmeli.
- Query fonksiyonlari backend contract ile birebir uyumlu olmali.

Bitis kriteri:

- Admin panel build verir.
- Admin ekranlarinda uydurma veri veya stale placeholder kalmaz.

### 5. Auth'i minimum internal seviyeye cikar

- tek admin kullaniciyi env veya seed ile tanimla
- hashed password kontrolu ekle
- mevcut JWT akisini koru
- refresh token'i ya kaldir ya da gercek lifecycle ile uygula

Bitis kriteri:

- Rastgele e-posta/sifre ile admin login alinmaz.

### 6. Analytics anlamini duzelt

- `total_badges` katalog mu kazanilmis badge mi, bunu netlestir ve tek tanim sec.
- `activeUsers` ve `totalRules` backend tarafinda gercekten hesaplanmiyorsa frontend'te uydurma veri kullanma.
- `points-history` period parametresini gercek aggregation'a bagla.

Bitis kriteri:

- Analytics ekranindaki butun sayilar backend tarafinda tanimli ve anlami net olur.

### 7. Operability kapanis maddeleri

- Kafka health check'i gercek probe yap.
- seed/reset/runbook dokumanlarini gercek calisma adimlariyla guncelle.
- Docker-first smoke checklist'ini tekrar calistir.

Bitis kriteri:

- Tek komutla sistem kalkar, test event basilir, sonuc admin'de gorulur ve bu akisin dokumani repo icinde gunceldir.

## Guncel Dogrulama Notlari

Su audit sirasinda gorulen net sinyaller:

- `docker compose build muscle` gecti.
- `npm run build` admin tarafinda gecmedi.
- Bu makinede `go` CLI gorunmedigi icin `go test ./...` dogrudan calistirilamadi.

Bu nedenle mevcut durum:

- backend tarafinda belirgin ilerleme var
- admin ve data consistency tarafi henuz kapanmamis
- plan `tamamlandi` olarak isaretlenemez

## Defaultlar

- Kapsam hala sadece `Backend + Admin`
- `mobile/` ilk faz disinda
- Mimari halen `Neo4j + Redis + Kafka`
- Hedef halen `ic kullanim beta`
- Bu dokumanda `eksik` veya `yanlis` denilen her madde mevcut kod audit'ine dayalidir; varsayim degildir
