# QSL management system end-to-end API verification (ASCII only, PS 5.1+ compatible)
# Usage: powershell -File scripts/e2e_verify.ps1  (backend :8000 + frontend :3000 must be running)
$Base  = 'http://127.0.0.1:8000'      # gin binds 0.0.0.0
$ViaFE = 'http://localhost:3000'      # vite binds localhost (may be ::1)
$script:pass = 0
$script:fail = 0

function Check($name, $cond, $detail = '') {
  if ($cond) { $script:pass++; Write-Output ("PASS  " + $name) }
  else { $script:fail++; Write-Output ("FAIL  " + $name + "  :: " + $detail) }
}

function Req($method, $uri, $body, $token) {
  $p = @{ Uri = $uri; Method = $method; UseBasicParsing = $true; TimeoutSec = 20; ErrorAction = 'Stop' }
  if ($body) { $p.Body = ($body | ConvertTo-Json -Depth 6); $p.ContentType = 'application/json; charset=utf-8' }
  if ($token) { $p.Headers = @{ Authorization = "Bearer $token" } }
  try {
    $resp = Invoke-WebRequest @p
    return @{ StatusCode = [int]$resp.StatusCode; Content = [string]$resp.Content }
  } catch {
    $sc = -1
    try { $sc = [int]$_.Exception.Response.StatusCode } catch { $sc = -1 }
    return @{ StatusCode = $sc; Content = '' }
  }
}

function D($r) { if ($r.Content) { return ($r.Content | ConvertFrom-Json) } else { return $null } }

# ---------- 0. wait for backend ----------
$ready = $false
for ($i = 0; $i -lt 40; $i++) {
  $probe = Req 'GET' "$Base/" $null $null
  if ($probe.StatusCode -eq 200) { $ready = $true; break }
  Start-Sleep -Milliseconds 750
}
if (-not $ready) { Write-Output "FATAL backend not ready on $Base"; exit 1 }
Write-Output "== backend ready =="

# ---------- 1. frontend pages (SPA fallback) ----------
$r = Req 'GET' "$ViaFE/" $null $null
Check "FE / 200" ($r.StatusCode -eq 200)
$r = Req 'GET' "$ViaFE/apply" $null $null
Check "FE /apply 200 (SPA fallback)" ($r.StatusCode -eq 200)
$r = Req 'GET' "$ViaFE/confirm" $null $null
Check "FE /confirm 200 (SPA fallback)" ($r.StatusCode -eq 200)
$r = Req 'GET' "$ViaFE/admin/cards" $null $null
Check "FE /admin/cards 200 (SPA fallback)" ($r.StatusCode -eq 200)

# ---------- 2. admin login ----------
$r = Req 'POST' "$Base/api/auth/login" @{ username = 'admin'; password = 'admin123' }
$tok = (D $r).data.access_token
Check "login 200 + token" ($r.StatusCode -eq 200 -and $tok)
if (-not $tok) { Write-Output "FATAL no token"; exit 1 }

# mute notify_email during the run: every submitted test application would
# otherwise send a real "new application" mail to the configured inbox;
# original value restored at the end (site_url/site_name untouched)
$script:notifySaved = $null
$r = Req 'GET' "$Base/api/settings/site" $null $tok
$site0 = D $r
if ($site0.data.notify_email) {
  $script:notifySaved = [string]$site0.data.notify_email
  Req 'POST' "$Base/api/settings/site" @{ site_name = [string]$site0.data.site_name; site_url = [string]$site0.data.site_url; notify_email = '' } $tok | Out-Null
  Check "notify email muted for E2E" ($true)
}

# ---------- 3. public submit: QSO application with reason ----------
$r = Req 'POST' "$Base/api/public/exchange-online" @{
  scene_type = 'QSO'; call_sign = 'E2EQSO'
  qso_date = '2024-11-05'; qso_time = '12:30'; qso_freq = '14.270'; qso_band = '20m'
  qso_mode = 'SSB'; qso_rst_sent = '59'; qso_rst_rcvd = '59'
  application_reason = 'E2E automated test exchange reason'
  email = 'e2e@example.com'; name = 'E2E Tester'; postal_code = '100000'
  address = 'Test Road 1, Beijing'; use_bureau = $false
}
$qso = D $r
Check "submit QSO 200" ($r.StatusCode -eq 200 -and $qso.data.scene_type -eq 'QSO')
Check "QSO request_code random EX+8" ($qso.data.request_code -match '^EX[23456789A-HJ-NP-Z]{8}$')
$qsoId = $qso.data.id
$exCode = $qso.data.request_code

# ---------- 4. validation negatives ----------
$r = Req 'POST' "$Base/api/public/exchange-online" @{
  scene_type = 'QSO'; call_sign = 'E2EBAD2'; qso_date = '2024-11-05'; qso_mode = 'SSB'
  email = 'e2e@example.com'; address = 'x'
}
Check "QSO missing freq -> 400" ($r.StatusCode -eq 400)
$r = Req 'POST' "$Base/api/public/exchange-online" @{
  scene_type = 'XYZ'; call_sign = 'E2EBAD2'; qso_date = '2024-11-05'; qso_freq = '14.270'; qso_mode = 'SSB'
  email = 'e2e@example.com'; address = 'x'
}
Check "invalid scene -> 400" ($r.StatusCode -eq 400)
$r = Req 'POST' "$Base/api/public/exchange-online" @{
  scene_type = 'SWL'; call_sign = 'E2EBAD'; swl_date = '2024-11-01'; swl_freq = '7.050'
  email = 'e2e@example.com'; address = 'x'
}
Check "SWL missing mode -> 400" ($r.StatusCode -eq 400)
$r = Req 'POST' "$Base/api/public/exchange-online" @{
  scene_type = 'EYEBALL'; call_sign = 'E2EBAD'; eyeball_date = '2024-11-01'
  email = 'e2e@example.com'; address = 'x'
}
Check "EYEBALL missing activity -> 400" ($r.StatusCode -eq 400)
$r = Req 'POST' "$Base/api/public/exchange-online" @{
  scene_type = 'QSO'; call_sign = 'E2EBAD'; qso_date = '2024-11-05'; qso_freq = '14.270'; qso_mode = 'SSB'
  use_bureau = $true; email = 'e2e@example.com'; address = 'x'
}
Check "bureau without name -> 400" ($r.StatusCode -eq 400)
$r = Req 'POST' "$Base/api/public/exchange-online" @{
  scene_type = 'QSO'; call_sign = 'E2EQSO'
  qso_date = '2024-11-05'; qso_freq = '14.270'; qso_mode = 'SSB'
  email = 'e2e@example.com'; address = 'x'
}
Check "duplicate pending call_sign -> 409" ($r.StatusCode -eq 409)

# ---------- 5. application_reason optional ----------
$r = Req 'POST' "$Base/api/public/exchange-online" @{
  scene_type = 'SWL'; call_sign = 'E2ESWL'; swl_date = '2024-11-02'; swl_freq = '7.050'
  swl_band = '40m'; swl_mode = 'FT8'; use_bureau = $true; bureau_name = 'E2E TEST BUREAU'
  email = 'e2e@example.com'; name = 'SWL'; address = 'x'
}
$swl = D $r; $swlId = $swl.data.id
Check "submit SWL without reason 200" ($r.StatusCode -eq 200 -and $swl.data.scene_type -eq 'SWL')
$r = Req 'POST' "$Base/api/public/exchange-online" @{
  scene_type = 'EYEBALL'; call_sign = 'E2EEYE'; eyeball_date = '2024-11-03'; eyeball_time = '14:00'
  eyeball_activity = 'E2E Hamfest'; eyeball_location = 'Test Hall'; email = 'e2e@example.com'
  eyeball_type = 'ONLINE'; name = 'EYE'; address = 'x'
}
$eye = D $r; $eyeId = $eye.data.id
Check "submit EYEBALL without reason 200 + online type" ($r.StatusCode -eq 200 -and $eye.data.scene_type -eq 'EYEBALL' -and $eye.data.eyeball_type -eq 'ONLINE')

# ---------- 6. admin list + scene filter ----------
$r = Req 'GET' "$Base/api/exchange/online/requests?scene_type=QSO&size=50" $null $tok
$lq = D $r
$nonQso = ($lq.data.items | Where-Object { $_.scene_type -ne 'QSO' } | Measure-Object).Count
$hasQso = ($lq.data.items | Where-Object { $_.id -eq $qsoId } | Measure-Object).Count
Check "admin list scene=QSO filter" ($r.StatusCode -eq 200 -and $nonQso -eq 0 -and $hasQso -eq 1)
$r = Req 'GET' "$Base/api/exchange/online/requests?scene_type=SWL&size=50" $null $tok
$nonSwl = ((D $r).data.items | Where-Object { $_.scene_type -ne 'SWL' } | Measure-Object).Count
Check "admin list scene=SWL filter" ($nonSwl -eq 0)

# ---------- 7. approve (+ReviewedBy) then create card ----------
$r = Req 'POST' "$Base/api/exchange/online/requests/$qsoId/approve" $null $tok
$ap = D $r
Check "approve 200 + reviewed_by=admin" ($r.StatusCode -eq 200 -and $ap.data.review_status -eq 'APPROVED' -and $ap.data.reviewed_by -eq 'admin')
$r = Req 'POST' "$Base/api/exchange/online/requests/$qsoId/approve" $null $tok
Check "re-approve -> 422" ($r.StatusCode -eq 422)
$r = Req 'POST' "$Base/api/exchange/online/requests/$qsoId/create-card" $null $tok
$cc = D $r
$cardCode = $cc.data.card_code
$cardId = $cc.data.id
Check "create card 200 + random code C+8" ($r.StatusCode -eq 200 -and $cardCode -match '^C[23456789A-HJ-NP-Z]{8}$')
Check "card scene mapping QSO" ($cc.data.card_type -eq 'QSO' -and $cc.data.scene_type -eq 'QSO')
$br = $cc.data.business_remarks
Check "business_remarks has reason+evidence" ($br -match 'E2E automated test' -and $br -match '2024-11-05' -and $br -match '14.270')
$r = Req 'POST' "$Base/api/exchange/online/requests/$qsoId/create-card" $null $tok
Check "re-create-card -> 422" ($r.StatusCode -eq 422)

# ---------- 8. card flow: issue -> sent ----------
$r = Req 'GET' "$Base/api/card-records?call_sign=E2EQSO" $null $tok
$cl = D $r
$card = $cl.data.items | Select-Object -First 1
Check "card list flow=PENDING_ISSUE" ($card.flow_status -eq 'PENDING_ISSUE')
$r = Req 'POST' "$Base/api/card-records/$cardId/issue" $null $tok
Check "issue -> ISSUED" ((D $r).data.flow_status -eq 'ISSUED')
$r = Req 'POST' "$Base/api/card-records/$cardId/sent" @{
  mail_type = 'REGISTERED'; tracking_number = 'E2E1234567890'; tracking_carrier = 'CHINA_POST'; sent_remarks = 'e2e'
} $tok
$sent = D $r
Check "sent -> SENT + tracking + mail status" ($sent.data.flow_status -eq 'SENT' -and $sent.data.card_sent -eq $true -and ($sent.data.sent_mail_status -eq 'SENT' -or $sent.data.sent_mail_status -eq 'FAILED') -and $sent.data.tracking_number -eq 'E2E1234567890')

# ---------- 9. public card lookup (ConfirmReceipt field regression) ----------
$r = Req 'GET' "$Base/api/public/cards/$cardCode" $null $null
$pc = D $r
$needKeys = @('card_code','call_sign','card_type','card_version','flow_status','mail_type','tracking_number','card_sent','card_received','receipt_confirmed','card_date')
$missing = $needKeys | Where-Object { -not ($pc.data.PSObject.Properties.Name -contains $_) }
Check "public card all fields present" (($missing | Measure-Object).Count -eq 0) ("missing: " + ($missing -join ','))
Check "public card state SENT/unconfirmed" ($pc.data.flow_status -eq 'SENT' -and $pc.data.card_sent -eq $true -and $pc.data.receipt_confirmed -eq $false)
Check "public card not found -> 404" ((Req 'GET' "$Base/api/public/cards/NOPE99" $null $null).StatusCode -eq 404)
$r = Req 'GET' "$Base/api/public/station-mail-info" $null $null
Check "public station mail info 200" ($r.StatusCode -eq 200)
# return address source: pick an address book entry, station-mail-info must follow it
$r = Req 'GET' "$Base/api/settings/site" $null $tok
$ra0 = (D $r).data.return_address_id
$ab = Req 'POST' "$Base/api/address/book" @{ call_sign = 'E2EAB'; name = 'E2E AB'; postal_code = '200000'; address = 'E2E ADDRESS BOOK LINE 1' } $tok
$abId = (D $ab).data.id
Req 'POST' "$Base/api/settings/site" @{ return_address_id = "$abId" } $tok | Out-Null
$r = Req 'GET' "$Base/api/public/station-mail-info" $null $null
$smi2 = D $r
Check "station-mail-info follows address book pick" ($smi2.data.address -eq 'E2E ADDRESS BOOK LINE 1')
Req 'POST' "$Base/api/settings/site" @{ return_address_id = "$ra0" } $tok | Out-Null
Req 'DELETE' "$Base/api/address/book/$abId" $null $tok | Out-Null
$r = Req 'GET' "$Base/api/public/stats" $null $null
$pubStats = D $r
Check "public stats 200 + fields" ($r.StatusCode -eq 200 -and $null -ne $pubStats.data.cards_sent -and $null -ne $pubStats.data.cards_signed -and $null -ne $pubStats.data.pending_requests)

# ---------- 10. frontend proxy ----------
$r = Req 'GET' "$ViaFE/api/public/cards/$cardCode" $null $null
Check "FE proxy /api -> :8000" ($r.StatusCode -eq 200 -and (D $r).data.card_code -eq $cardCode)

# ---------- 11. public confirm receipt loop ----------
$r = Req 'POST' "$Base/api/public/confirm-receipt" @{ card_code = $cardCode; received_date = '2024-11-20'; remarks = 'E2E signed' }
$cr = D $r
Check "confirm-receipt 200 + SIGNED (no receive record)" ($r.StatusCode -eq 200 -and $cr.data.flow_status -eq 'SIGNED' -and (-not $cr.data.receive_code))
$r = Req 'POST' "$Base/api/public/confirm-receipt" @{ card_code = $cardCode; call_sign = 'WRONG99'; received_date = '2024-11-20' }
Check "confirm wrong callsign -> 400" ($r.StatusCode -eq 400)
$r = Req 'GET' "$Base/api/public/cards/$cardCode" $null $null
$pc2 = D $r
Check "after confirm: SIGNED + confirmed" ($pc2.data.flow_status -eq 'SIGNED' -and $pc2.data.receipt_confirmed -eq $true -and $pc2.data.card_received -eq $true)

# return-mail registration: admin enables per-card, opponent registers, admin confirms receiving
$r = Req 'POST' "$Base/api/card-records/$cardId/return-toggle" @{ enabled = $true } $tok
Check "return-toggle on 200" ($r.StatusCode -eq 200 -and (D $r).data.return_mail_enabled -eq $true)
$r = Req 'POST' "$Base/api/public/return-mail" @{ card_code = $cardCode; mail_type = 'REGISTERED'; tracking_number = 'E2ERT0000001' }
Check "return-mail register 200" ($r.StatusCode -eq 200 -and (D $r).data.return_tracking -eq 'E2ERT0000001')
$r = Req 'POST' "$Base/api/public/return-mail" @{ card_code = $cardCode; mail_type = 'REGISTERED'; tracking_number = '' }
Check "return-mail registered w/o tracking -> 400" ($r.StatusCode -eq 400)
$r = Req 'GET' "$Base/api/public/cards/$cardCode" $null $null
$pc3 = D $r
Check "card shows return mail fields" ($pc3.data.return_mailed_at -ne '' -and $pc3.data.return_mail_type -eq 'REGISTERED' -and $pc3.data.return_mail_enabled -eq $true)
$r = Req 'POST' "$Base/api/card-records/$cardId/return-receive" $null $tok
$retR = D $r
Check "return-receive 200 + receive code" ($r.StatusCode -eq 200 -and $retR.data.receive_code -match '^R\d{4}-\d{8}$')
$r = Req 'GET' "$Base/api/receive-records?size=10" $null $tok
$retList = D $r
$retHit = $false
foreach ($it in $retList.data.items) { if ($it.receive_code -eq $retR.data.receive_code) { $retHit = $true } }
Check "return receipt auto in receive records" ($retHit -eq $true)
$r = Req 'POST' "$Base/api/card-records/$cardId/return-receive" $null $tok
Check "return-receive repeat -> 422" ($r.StatusCode -eq 422)
$r = Req 'POST' "$Base/api/card-records/$cardId/return-toggle" @{ enabled = $false } $tok
$r = Req 'POST' "$Base/api/public/return-mail" @{ card_code = $cardCode; mail_type = 'ORDINARY'; tracking_number = '' }
Check "return-mail when disabled -> 422" ($r.StatusCode -eq 422)
$r = Req 'POST' "$Base/api/card-records/$cardId/return-toggle" @{ enabled = $true } $tok
Check "return-toggle re-enable 200" ($r.StatusCode -eq 200)

# ---------- 12. SWL / EYEBALL card mapping ----------
Req 'POST' "$Base/api/exchange/online/requests/$swlId/approve" $null $tok | Out-Null
$r = Req 'POST' "$Base/api/exchange/online/requests/$swlId/create-card" $null $tok
$swlCard = D $r
Check "SWL card mapping + evidence + bureau" ($r.StatusCode -eq 200 -and $swlCard.data.card_type -eq 'SWL' -and $swlCard.data.business_remarks -match '2024-11-02' -and $swlCard.data.business_remarks -match '7.050' -and $swlCard.data.business_remarks -match 'E2E TEST BUREAU')
Req 'POST' "$Base/api/exchange/online/requests/$eyeId/approve" $null $tok | Out-Null
$r = Req 'POST' "$Base/api/exchange/online/requests/$eyeId/create-card" $null $tok
$eyeCard = D $r
Check "EYEBALL card mapping + evidence" ($r.StatusCode -eq 200 -and $eyeCard.data.card_type -eq 'EYEBALL' -and $eyeCard.data.business_remarks -match '2024-11-03' -and $eyeCard.data.business_remarks -match 'E2E Hamfest')

# ---------- 13. request status endpoint + public page ----------
$r = Req 'GET' "$Base/api/public/exchange-status/$exCode" $null $null
$st = D $r
Check "exchange-status 200 + SIGNED" ($r.StatusCode -eq 200 -and $st.data.request_code -eq $exCode -and $st.data.card_created -eq $true -and $st.data.flow_status -eq 'SIGNED')
Check "exchange-status not found -> 404" ((Req 'GET' "$Base/api/public/exchange-status/EXNOPE999" $null $null).StatusCode -eq 404)
$r = Req 'GET' "$ViaFE/status/$exCode" $null $null
Check "FE /status/:code 200 (SPA fallback)" ($r.StatusCode -eq 200)

# ---------- 14. image upload + static serving ----------
$b64 = 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=='
$b64file = Join-Path $PSScriptRoot 'img_b64.txt'
Set-Content -Path $b64file -Value $b64 -Encoding Ascii
$tmpImg = Join-Path $PSScriptRoot 'test_image.png'
certutil -decode $b64file $tmpImg | Out-Null
$resp = curl.exe -s -F "file=@$tmpImg" -H "Authorization: Bearer $tok" "$Base/api/upload/image"
$up = $resp | ConvertFrom-Json
Check "upload image 200 + /uploads/ url" ($up.data.url -like '/uploads/*')
$sc = curl.exe -s -o NUL -w "%{http_code}" "$Base$($up.data.url)"
Check "uploaded file served 200" ($sc -eq '200')
$tmpTxt = Join-Path $PSScriptRoot 'test_file.txt'
Set-Content -Path $tmpTxt -Value 'not an image' -Encoding Ascii
$sc2 = curl.exe -s -o NUL -w "%{http_code}" -F "file=@$tmpTxt" -H "Authorization: Bearer $tok" "$Base/api/upload/image"
Check "non-image upload -> 400" ($sc2 -eq '400')
$sc3 = curl.exe -s -o NUL -w "%{http_code}" -F "file=@$tmpImg" "$Base/api/upload/image"
Check "upload without token -> 401" ($sc3 -eq '401')

# ---------- 15. removed public query endpoints ----------
Check "removed qso-query -> 404" ((Req 'GET' "$Base/api/public/qso-query?call_sign=XXX" $null $null).StatusCode -eq 404)
Check "removed card-query -> 404" ((Req 'GET' "$Base/api/public/card-query?call_sign=XXX" $null $null).StatusCode -eq 404)

# ---------- 16. tracking fixes ----------
Check "tracking not found -> 404" ((Req 'GET' "$Base/api/public/tracking?tracking_number=NOPE123" $null $null).StatusCode -eq 404)
$r = Req 'GET' "$Base/api/public/tracking?tracking_number=E2E1234567890" $null $null
$tr = D $r
Check "tracking found + fields" ($r.StatusCode -eq 200 -and $tr.data.tracking_number -eq 'E2E1234567890' -and $tr.data.auto -eq $false)
$r = Req 'GET' "$Base/api/public/exchange-status/$exCode" $null $null
$st2 = D $r
Check "exchange-status has tracking block" ($r.StatusCode -eq 200 -and ($st2.data.PSObject.Properties.Name -contains 'tracking_details'))

# ---------- 17. password change + site settings ----------
$r = Req 'GET' "$Base/api/public/site-info" $null $null
$defaultSite = (D $r).data.site_name
Check "public site-info 200 + name" ($r.StatusCode -eq 200 -and $defaultSite)
$r = Req 'POST' "$Base/api/auth/change-password" @{ old_password = 'wrongpass'; new_password = 'e2enewpass123' } $tok
Check "change-password wrong old -> 400" ($r.StatusCode -eq 400)
$r = Req 'POST' "$Base/api/auth/change-password" @{ old_password = 'admin123'; new_password = '123' } $tok
Check "change-password short new -> 400" ($r.StatusCode -eq 400)
$r = Req 'POST' "$Base/api/auth/change-password" @{ old_password = 'admin123'; new_password = 'e2enewpass123' } $null
Check "change-password without token -> 401" ($r.StatusCode -eq 401)
$r = Req 'GET' "$Base/api/settings/site" $null $tok
Check "settings/site GET 200" ($r.StatusCode -eq 200)
$r = Req 'POST' "$Base/api/settings/site" @{ site_name = 'E2E-SITE-A' } $tok
Check "settings/site save 200" ($r.StatusCode -eq 200)
$r = Req 'POST' "$Base/api/settings/site" @{ site_name = 'E2E-SITE-B' } $tok
$r = Req 'GET' "$Base/api/public/site-info" $null $null
Check "site save repeatable (upsert)" ($r.StatusCode -eq 200 -and (D $r).data.site_name -eq 'E2E-SITE-B')
$r = Req 'POST' "$Base/api/settings/site" @{ site_name = $defaultSite } $tok
$r = Req 'GET' "$Base/api/public/site-info" $null $null
Check "site restored -> original name" ($r.StatusCode -eq 200 -and (D $r).data.site_name -eq $defaultSite)

# site notice (announcement board) save + public visibility + sender email field
$r = Req 'GET' "$Base/api/settings/site" $null $tok
$notice0 = [string]((D $r).data.site_notice)
$r = Req 'POST' "$Base/api/settings/site" @{ site_notice = 'E2E NOTICE BOARD' } $tok
$r = Req 'GET' "$Base/api/public/site-info" $null $null
$siN = D $r
$hasSender = $siN.data.PSObject.Properties.Name -contains 'sender_email'
Check "site notice public + sender_email field" ($siN.data.site_notice -eq 'E2E NOTICE BOARD' -and $hasSender)
Req 'POST' "$Base/api/settings/site" @{ site_notice = $notice0 } $tok | Out-Null
Check "site notice restored" ($true)

# ---------- 18. QSO record_code sequence monotonic ----------
$r1 = Req 'POST' "$Base/api/qso-records" @{ call_sign = 'E2ESEQX'; date = '2025-01-01'; scene_type = 'QSO' } $tok
$r2 = Req 'POST' "$Base/api/qso-records" @{ call_sign = 'E2ESEQY'; date = '2025-01-01'; scene_type = 'QSO' } $tok
$qa = (D $r1).data; $qb = (D $r2).data
Check "qso record_code sequence distinct" ($r1.StatusCode -eq 200 -and $r2.StatusCode -eq 200 -and $qa.record_code -and $qb.record_code -and ($qa.record_code -ne $qb.record_code))

# ---------- 19. ADIF import (unique callsign per run) ----------
$adiSuffix = ('{0:X3}' -f (Get-Random -Maximum 4096))
$adiCall = 'E2EADI' + $adiSuffix
$adi = '<CALL:9>' + $adiCall + '<QSO_DATE:8>20250203<TIME_ON:6>101010<FREQ:6>14.074<BAND:3>20m<MODE:4>FT8<RST_SENT:2>59<RST_RCVD:2>59<EOR>'
$adiFile = Join-Path $PSScriptRoot 'test_adif.adi'
Set-Content -Path $adiFile -Value $adi -Encoding Ascii
$imp = curl.exe -s -F "file=@$adiFile" -H "Authorization: Bearer $tok" "$Base/api/qso-records/import"
$impD = $imp | ConvertFrom-Json
Check "adif import 200 + imported 1" ($impD.data.imported -eq 1 -and $impD.data.skipped -eq 0)
$imp2 = curl.exe -s -F "file=@$adiFile" -H "Authorization: Bearer $tok" "$Base/api/qso-records/import"
$imp2D = $imp2 | ConvertFrom-Json
Check "adif re-import dedup skipped" ($imp2D.data.imported -eq 0 -and $imp2D.data.skipped -eq 1)

# ---------- 20. SWL return-mail flow (he mails first) ----------
# address book entry must exist BEFORE approve: approval auto-sends return address
$r = Req 'POST' "$Base/api/address/book" @{ name = 'E2E MyAddr'; address = '1 Test Road'; postal_code = '100000'; destination_country = 'China' } $tok
$myAddr = D $r
Check "address create 200" ($r.StatusCode -eq 200 -and $myAddr.data.id)
$r = Req 'POST' "$Base/api/public/exchange-online" @{ scene_type = 'SWL'; call_sign = 'E2ESWLRT'; swl_date = '2025-02-03'; swl_time = '19:42'; swl_freq = '7.074'; swl_band = '40m'; swl_mode = 'FT8'; application_reason = 'e2e swl return flow'; email = 'e2e@example.com'; name = 'E2E SWL'; postal_code = '100000'; address = 'E2E ReturnAddr 1-2-3' }
$swlrt = D $r
Check "swl submit 200 + address + time" ($r.StatusCode -eq 200 -and $swlrt.data.request_code -and $swlrt.data.swl_time -eq '19:42' -and $swlrt.data.address -match 'E2E ReturnAddr')
$swlrtId = $swlrt.data.id
$swlrtCode = $swlrt.data.request_code
$r = Req 'POST' "$Base/api/exchange/online/requests/$swlrtId/approve" $null $tok
$ap = D $r
Check "swl approve 200" ($r.StatusCode -eq 200)
Check "swl approve auto address sent (single entry)" ($ap.data.return_address_text -and $ap.data.address_sent_at -and ($ap.data.return_address_text -notmatch '地址 2'))
$r = Req 'POST' "$Base/api/public/exchange-return-mail" @{ request_code = $swlrtCode; mail_type = 'REGISTERED'; tracking_number = 'E2ERT987654321' } $null
Check "return-mail register 200" ($r.StatusCode -eq 200)
$r = Req 'POST' "$Base/api/public/exchange-return-mail" @{ request_code = $swlrtCode; mail_type = 'REGISTERED'; tracking_number = '' } $null
Check "return-mail registered no number -> 400" ($r.StatusCode -eq 400)
$r = Req 'GET' "$Base/api/public/exchange-status/$swlrtCode" $null $null
$stR = D $r
Check "exchange-status has return fields" ($stR.data.return_mail_type -eq 'REGISTERED' -and $stR.data.return_tracking -eq 'E2ERT987654321')

# ---------- 21. re-send my addresses to applicant (approval already auto-sent) ----------
$r = Req 'POST' "$Base/api/exchange/online/requests/$swlrtId/send-address" @{ address_ids = @($myAddr.data.id) } $tok
$sa = D $r
Check "send-address 200 + text stored" ($r.StatusCode -eq 200 -and ($sa.data.address_text -match 'E2E MyAddr'))
$r = Req 'GET' "$Base/api/public/exchange-status/$swlrtCode" $null $null
$stR2 = D $r
Check "return_address_text public visible" ($stR2.data.return_address_text -match 'E2E MyAddr')

# ---------- 22. from-qso card creation + real smtp test guard ----------
$r = Req 'POST' "$Base/api/qso-records" @{ call_sign = 'E2EFQSO1'; date = '2025-02-05'; scene_type = 'QSO'; freq = '21.300'; mode = 'SSB' } $tok
$fq = D $r
Check "qso create for from-qso 200" ($r.StatusCode -eq 200 -and $fq.data.id)
$r = Req 'POST' "$Base/api/card-records/from-qso" @{ qso_record_id = $fq.data.id; card_version = ''; mail_type = 'REGISTERED' } $tok
$fcard = D $r
Check "from-qso create card 200 + fields" ($r.StatusCode -eq 200 -and $fcard.data.card_code -and $fcard.data.call_sign -eq 'E2EFQSO1' -and $fcard.data.scene_type -eq 'QSO')
$r = Req 'POST' "$Base/api/card-records/from-qso" @{ qso_record_id = $fq.data.id } $tok
Check "from-qso re-create -> 422" ($r.StatusCode -eq 422)
$r = Req 'GET' "$Base/api/qso-records/$($fq.data.id)" $null $tok
Check "qso has_card set true" ((D $r).data.has_card -eq $true)
$r = Req 'POST' "$Base/api/settings/smtp/test?to_email=e2e@example.com" $null $tok
Check "smtp test endpoint responsive (200 ok/422 unconf/500 fail/timeout)" ($r.StatusCode -eq 200 -or $r.StatusCode -eq 422 -or $r.StatusCode -eq 500 -or $r.StatusCode -eq -1)
$r = Req 'POST' "$Base/api/settings/smtp/test" $null $tok
Check "smtp test no email -> 400" ($r.StatusCode -eq 400)

# ---------- 23. SWL receive-return (inbound card -> receive records) ----------
$r = Req 'POST' "$Base/api/exchange/online/requests/$swlrtId/receive-return" @{ remarks = 'e2e received ok' } $tok
$rr = D $r
Check "receive-return 200 + receive_code" ($r.StatusCode -eq 200 -and ($rr.data.receive_code -match '^R\d{4}-'))
Check "receive-return auto card created" ($rr.data.card_code -match '^C')
Check "receive-return re-post -> 422" ((Req 'POST' "$Base/api/exchange/online/requests/$swlrtId/receive-return" $null $tok).StatusCode -eq 422)
$r = Req 'GET' "$Base/api/public/exchange-status/$swlrtCode" $null $null
$stR3 = D $r
Check "exchange-status return_received_at" ($stR3.data.return_received_at)
Check "exchange-status auto card flow" ($stR3.data.card_created -eq $true -and $stR3.data.flow_status -eq 'PENDING_ISSUE')
$r = Req 'GET' "$Base/api/receive-records?page=1&size=5" $null $tok
$recs = (D $r).data.items
Check "receive-records has swl inbound first" ($recs[0].receive_code -eq $rr.data.receive_code -and $recs[0].business_type -eq 'SWL')

# ---------- 24. reject with reason (+ email best-effort) + data export ----------
$r = Req 'POST' "$Base/api/public/exchange-online" @{ scene_type = 'QSO'; call_sign = 'E2ERJCT'; qso_date = '2025-02-06'; qso_freq = '14.200'; qso_mode = 'SSB'; application_reason = 'e2e reject flow'; email = 'e2e@example.com'; name = 'E2E RJ'; postal_code = '100000'; address = 'E2E RJ Addr' }
$rj = D $r
$r = Req 'POST' "$Base/api/exchange/online/requests/$($rj.data.id)/reject" @{ review_reason = 'e2e reject reason' } $tok
$rjd = D $r
Check "reject 200 + status + reason" ($r.StatusCode -eq 200 -and $rjd.data.review_status -eq 'REJECTED' -and $rjd.data.review_reason -eq 'e2e reject reason')
$r = Req 'POST' "$Base/api/exchange/online/requests/$($rj.data.id)/reject" @{ review_reason = 'x' } $tok
Check "reject re-reject -> 422" ($r.StatusCode -eq 422)
$r = Req 'GET' "$Base/api/admin/export" $null $tok
$exp = D $r
Check "admin export 200 + tables" ($r.StatusCode -eq 200 -and $exp.tables -and $exp.tables.exchange_requests -and $exp.tables.card_records)

# ---------- 25. admin import (restore from the export above) ----------
$expFile = Join-Path $env:TEMP "qsl-e2e-backup.json"
[System.IO.File]::WriteAllText($expFile, $r.Content, (New-Object System.Text.UTF8Encoding($false)))
$impRaw = curl.exe -s -F "file=@$expFile" -H "Authorization: Bearer $tok" "$Base/api/admin/import"
$impD = $impRaw | ConvertFrom-Json
Check "admin import 200 + restored rows" ($r.StatusCode -eq 200 -and $impD.data.restored -and $impD.data.restored.qso_records -gt 0 -and $impD.data.restored.system_settings -gt 0)
$r = Req 'GET' "$Base/api/exchange/online/requests?scene_type=SWL&size=1" $null $tok
Check "data alive after import" ($r.StatusCode -eq 200 -and ((D $r).data.total -gt 0))
Write-Output ""
Write-Output ("RESULT: PASS=" + $script:pass + "  FAIL=" + $script:fail)

# restore notify_email muted at login (before result exit codes)
if ($script:notifySaved) {
  $r = Req 'GET' "$Base/api/settings/site" $null $tok
  $siteNow = D $r
  Req 'POST' "$Base/api/settings/site" @{ site_name = [string]$siteNow.data.site_name; site_url = [string]$siteNow.data.site_url; notify_email = $script:notifySaved } $tok | Out-Null
  Check "notify email restored" ($true)
}
if ($script:fail -gt 0) { exit 2 } else { exit 0 }
