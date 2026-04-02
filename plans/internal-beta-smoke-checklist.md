# Internal Beta Smoke Checklist

Bu checklist, iç beta sürümünü açmadan önce minimum operasyonel doğrulamayı standardize eder.

## Önkoşullar

- Repo kökünde güncel `.env` dosyası bulunur.
- Docker Desktop veya eşdeğer Docker Engine çalışır.
- Gerekliyse stack şu komutla kaldırılır:

```powershell
docker compose up -d neo4j redis zookeeper kafka muscle admin
```

- Admin varsayılan hesabı:
  - kullanıcı: `.env` içindeki `ADMIN_USERNAME`
  - şifre: `admin123` veya `.env` içinde hash’i üretilen karşılık

## Otomatik Smoke

Tercih edilen yol:

```powershell
./scripts/internal-beta-smoke.ps1 -StartStack
```

Script aşağıdaki doğrulamaları yapar:
- `GET /health`
- admin UI `200 OK`
- `POST /api/v1/auth/login`
- `GET /api/v1/users`
- test badge oluşturma
- test rule oluşturma
- `POST /api/v1/events/test` dry-run
- `POST /api/v1/events/test` execute
- kullanıcı puan/badge değişimi
- analytics summary/activity
- cleanup: puan geri alma, rule silme, badge silme

## Manuel Kontrol

Script dışında aşağıdaki görsel akış da doğrulanmalı:

1. `http://localhost:5173` açılır.
2. Admin hesabı ile giriş yapılır.
3. `Users` ekranında kullanıcı listesi görünür.
4. Bir kullanıcının profilinde `points`, `recent activity` ve `rich badge info` görüntülenir.
5. `Badges` ekranında create/edit/delete çalışır.
6. `Rules` ekranında create/edit/delete çalışır.
7. `Rules` ekranındaki test event modalı ile:
   - dry-run sonucu aksiyonları gösterir
   - execute sonucu kullanıcı profiline puan ve badge yazar
8. `Dashboard` ekranı backend error durumunda mock veri göstermeden gerçek state ile açılır.
9. `Analytics` ekranı summary, activity ve points history verilerini backend’den gösterir.

## Çıkış Kriteri

İç beta için minimum kabul durumu:
- `npm run build` geçer.
- `docker compose build muscle` geçer.
- `go test ./...` geçer.
- smoke script başarısız olmadan tamamlanır.
- manuel kontrol akışında kritik UI/API sapması görülmez.
